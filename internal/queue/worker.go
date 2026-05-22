package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/wa-server/internal/agent"
	"github.com/wa-server/internal/api/webhook"
	"github.com/wa-server/internal/metrics"
	"github.com/wa-server/internal/models"
	phonelib "github.com/wa-server/internal/phone"
	"github.com/wa-server/internal/repository"
)

type WhatsAppClient interface {
	SendMessage(ctx context.Context, to, messageType, content, mediaURL string) (string, error)
	SendMessageFromPhone(ctx context.Context, phoneNumberID, to, messageType, content, mediaURL string) (string, error)
	SendTemplateMessage(ctx context.Context, to, templateID string, params map[string]string) (string, error)
	SendTemplateMessageFromPhone(ctx context.Context, phoneNumberID, to, templateID string, params map[string]string) (string, error)
	GetPhoneNumbers(ctx context.Context) ([]models.WhatsAppPhoneNumber, error)
}

type CompanyRepoForWorker interface {
	GetByID(ctx context.Context, id string) (*models.Company, error)
	TryIncrementQuota(ctx context.Context, id string, amount int) (bool, error)
	DecrementQuota(ctx context.Context, id string, amount int) error
}

type BillingRepoForWorker interface {
	Create(ctx context.Context, log *models.BillingLog) error
}

type ConversationRepoForWorker interface {
	GetByID(ctx context.Context, id string) (*models.Conversation, error)
}

type PhoneNumberRepoForWorker interface {
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.PhoneNumber, error)
	GetByConversationID(ctx context.Context, conversationID string) (*models.PhoneNumber, error)
}

type WorkerPool struct {
	rmq             *RabbitMQ
	whatsapp        WhatsAppClient
	msgRepo         models.MessageRepository
	contactRepo     *repository.ContactRepository
	companyRepo     CompanyRepoForWorker
	billingRepo     BillingRepoForWorker
	convRepo        ConversationRepoForWorker
	phoneNumberRepo PhoneNumberRepoForWorker
	workers         int
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
	metrics         *metrics.Metrics
	agentTrackers   []*agent.Tracker
	wsHub           *webhook.WebSocketHub
}

func NewWorkerPool(
	rmq *RabbitMQ,
	whatsapp WhatsAppClient,
	msgRepo models.MessageRepository,
	contactRepo *repository.ContactRepository,
	companyRepo CompanyRepoForWorker,
	billingRepo BillingRepoForWorker,
	convRepo ConversationRepoForWorker,
	phoneNumberRepo PhoneNumberRepoForWorker,
	workers int,
	wsHub *webhook.WebSocketHub,
) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		rmq:             rmq,
		whatsapp:        whatsapp,
		msgRepo:         msgRepo,
		contactRepo:     contactRepo,
		companyRepo:     companyRepo,
		billingRepo:     billingRepo,
		convRepo:        convRepo,
		phoneNumberRepo: phoneNumberRepo,
		workers:         workers,
		ctx:             ctx,
		cancel:          cancel,
		wsHub:           wsHub,
	}
}

func (wp *WorkerPool) WithMetrics(m *metrics.Metrics) *WorkerPool {
	wp.metrics = m
	return wp
}

func (wp *WorkerPool) WithAgentTrackers(trackers []*agent.Tracker) *WorkerPool {
	wp.agentTrackers = trackers
	return wp
}

func (wp *WorkerPool) Start() error {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	if wp.metrics != nil {
		wp.metrics.SetWorkerActive(wp.workers)
	}
	slog.Info("worker pool started", "workers", wp.workers)
	return nil
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
	if wp.metrics != nil {
		wp.metrics.SetWorkerActive(0)
	}
	for _, t := range wp.agentTrackers {
		t.Stop()
	}
	slog.Info("worker pool stopped")
}

func (wp *WorkerPool) worker(id int) {
	tracker := wp.trackerFor(id)
	if tracker != nil {
		tracker.Start(wp.ctx)
		defer tracker.SetIdle()
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("worker panic recovered", "worker_id", id, "panic", r)
		}
	}()
	defer wp.wg.Done()

	slog.Info("worker starting", "worker_id", id)

	msgs, err := wp.rmq.Consume(wp.ctx, QueueOutbound)
	if err != nil {
		slog.Error("worker failed to start consumer", "worker_id", id, "error", err)
		return
	}

	slog.Info("worker started and consuming", "worker_id", id)

	for {
		select {
		case <-wp.ctx.Done():
			slog.Info("worker stopping", "worker_id", id)
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("channel closed, worker exiting", "worker_id", id)
				return
			}
			slog.Info("worker got message", "worker_id", id, "body_len", len(msg.Body))
			if tracker != nil {
				tracker.SetRunning()
			}
			wp.processMessage(wp.ctx, msg.Body, id)
			if err := msg.Ack(false); err != nil {
				slog.Error("failed to ack message", "error", err, "worker_id", id)
			}
			if tracker != nil {
				tracker.SetIdle()
			}
		}
	}
}

func (wp *WorkerPool) trackerFor(id int) *agent.Tracker {
	if id < len(wp.agentTrackers) {
		return wp.agentTrackers[id]
	}
	return nil
}

func (wp *WorkerPool) processMessage(ctx context.Context, body []byte, workerID int) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("processMessage panic", "panic", r)
		}
	}()

	slog.Info("worker received message", "body_length", len(body))

	var message struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		MessageType    string `json:"message_type"`
		Content        string `json:"content"`
		MediaURL       string `json:"media_url"`
		TemplateID     string `json:"template_id"`
		TemplateParams string `json:"template_params"`
		CompanyID      string `json:"company_id"`
	}

	if err := json.Unmarshal(body, &message); err != nil {
		slog.Error("failed to unmarshal message", "error", err)
		return
	}

	slog.Info("processing outbound message", "message_id", message.ID, "conversation_id", message.ConversationID)

	phone, err := wp.contactRepo.GetPhoneByConversationID(ctx, message.ConversationID)
	if err != nil {
		slog.Error("failed to get phone number", "error", err, "conversation_id", message.ConversationID)
		wp.failMessage(ctx, message.ID, "phone number not found")
		wp.incFailed(message.ID, workerID)
		return
	}

	slog.Info("resolved phone number", "phone", phone)
	phone = phonelib.Normalize(phone)

	pn, err := wp.phoneNumberRepo.GetByConversationID(ctx, message.ConversationID)
	if err != nil {
		slog.Error("failed to get phone number record", "error", err, "phone", phone)
		wp.failMessage(ctx, message.ID, "phone number not registered")
		wp.incFailed(message.ID, workerID)
		return
	}

	companyID := pn.CompanyID
	slog.Info("resolved company_id for broadcast", "company_id", companyID, "conversation_id", message.ConversationID)

	if wp.companyRepo != nil && companyID != "" {
		ok, err := wp.companyRepo.TryIncrementQuota(ctx, companyID, 1)
		if err != nil {
			slog.Error("quota check failed", "error", err, "company_id", companyID)
			wp.failMessage(ctx, message.ID, "internal error")
			wp.incFailed(message.ID, workerID)
			return
		}
		if !ok {
			slog.Warn("quota exceeded", "company_id", companyID, "message_id", message.ID)
			wp.failMessage(ctx, message.ID, "quota exceeded")
			wp.incFailed(message.ID, workerID)
			return
		}
	}

	var waMessageID string
	var sendErr error

	if message.MessageType == "template" {
		params := parseTemplateParams(message.TemplateParams)
		waMessageID, sendErr = wp.whatsapp.SendTemplateMessageFromPhone(ctx, pn.PhoneNumberID, phone, message.TemplateID, params)
	} else {
		waMessageID, sendErr = wp.whatsapp.SendMessageFromPhone(ctx, pn.PhoneNumberID, phone, message.MessageType, message.Content, message.MediaURL)
	}

	if sendErr != nil {
		slog.Error("failed to send message to WhatsApp", "error", sendErr, "message_id", message.ID)

		if wp.companyRepo != nil && companyID != "" {
			if err := wp.companyRepo.DecrementQuota(ctx, companyID, 1); err != nil {
				slog.Error("failed to decrement quota on failure", "error", err)
			}
		}

		wp.failMessage(ctx, message.ID, sendErr.Error())
		wp.incFailed(message.ID, workerID)
		return
	}

	if wp.msgRepo != nil {
		if err := wp.msgRepo.UpdateWAMessageID(ctx, message.ID, waMessageID); err != nil {
			slog.Error("failed to update WA message ID", "error", err, "message_id", message.ID)
		}
	}

	if wp.wsHub != nil {
		full, err := wp.msgRepo.GetByID(ctx, message.ID)
		if err == nil {
			full.Status = "sent"
			full.MessageID = waMessageID
			wp.wsHub.BroadcastToCompany(companyID, webhook.WebSocketMessage{
				Type:    "MessageStatusUpdated",
				Payload: full,
			})
			slog.Info("broadcast sent status via worker", "message_id", message.ID, "company_id", companyID)
		} else {
			slog.Error("failed to get message for broadcast", "error", err, "message_id", message.ID)
		}
	} else {
		slog.Warn("wsHub is nil, skipping status broadcast", "message_id", message.ID)
	}

	if wp.billingRepo != nil {
		category := "service"
		if message.MessageType == "template" {
			category = "marketing"
		}

		billingLog := &models.BillingLog{
			CompanyID:            companyID,
			ConversationID:       message.ConversationID,
			MessageID:            message.ID,
			PhoneNumber:          phone,
			ConversationCategory: category,
			CreatedAt:            time.Now().UTC(),
		}

		if err := wp.billingRepo.Create(ctx, billingLog); err != nil {
			slog.Error("failed to create billing log", "error", err, "message_id", message.ID)
		}
	}

	if wp.metrics != nil {
		wp.metrics.IncMessagesSent("success", "whatsapp")
	}

	tracker := wp.trackerFor(workerID)
	if tracker != nil {
		tracker.IncMessagesSent()
	}

	slog.Info("message sent successfully", "message_id", message.ID, "wa_message_id", waMessageID)
}

func (wp *WorkerPool) incFailed(messageID string, workerID int) {
	if wp.metrics != nil {
		wp.metrics.IncMessagesFailed()
	}
	tracker := wp.trackerFor(workerID)
	if tracker != nil {
		tracker.IncMessagesFailed()
	}
}

func (wp *WorkerPool) failMessage(ctx context.Context, messageID, errMsg string) {
	if wp.msgRepo != nil {
		if err := wp.msgRepo.SetFailed(ctx, messageID, errMsg); err != nil {
			slog.Error("failed to set message status to failed", "error", err)
		}
	}
}

func parseTemplateParams(params string) map[string]string {
	if params == "" {
		return nil
	}
	var result map[string]string
	_ = json.Unmarshal([]byte(params), &result)
	return result
}
