package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/repository"
)

type WhatsAppClient interface {
	SendMessage(ctx context.Context, to, messageType, content, mediaURL string) (string, error)
	SendMessageFromPhone(ctx context.Context, phoneNumberID, to, messageType, content, mediaURL string) (string, error)
	SendTemplateMessage(ctx context.Context, to, templateID string, params map[string]string) (string, error)
	SendTemplateMessageFromPhone(ctx context.Context, phoneNumberID, to, templateID string, params map[string]string) (string, error)
	GetPhoneNumbers(ctx context.Context) ([]models.WhatsAppPhoneNumber, error)
}

// CompanyRepoForWorker isolates company methods needed by the worker.
type CompanyRepoForWorker interface {
	GetByID(ctx context.Context, id string) (*models.Company, error)
	TryIncrementQuota(ctx context.Context, id string, amount int) (bool, error)
	DecrementQuota(ctx context.Context, id string, amount int) error
}

// BillingRepoForWorker isolates billing methods needed by the worker.
type BillingRepoForWorker interface {
	Create(ctx context.Context, log *models.BillingLog) error
}

// ConversationRepoForWorker isolates conversation methods needed by the worker.
type ConversationRepoForWorker interface {
	GetByID(ctx context.Context, id string) (*models.Conversation, error)
}

type PhoneNumberRepoForWorker interface {
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.PhoneNumber, error)
}

// WorkerPool manages concurrent RabbitMQ consumers for message processing.
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
}

// NewWorkerPool creates a worker pool with the specified number of concurrent workers.
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
	}
}

func (wp *WorkerPool) Start() error {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	slog.Info("worker pool started", "workers", wp.workers)
	return nil
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
	slog.Info("worker pool stopped")
}

func (wp *WorkerPool) worker(id int) {
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
			wp.processMessage(wp.ctx, msg.Body)
			if err := msg.Ack(false); err != nil {
				slog.Error("failed to ack message", "error", err, "worker_id", id)
			}
		}
	}
}

func (wp *WorkerPool) processMessage(ctx context.Context, body []byte) {
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
		return
	}

	slog.Info("resolved phone number", "phone", phone)

	pn, err := wp.phoneNumberRepo.GetByPhoneNumber(ctx, phone)
	if err != nil {
		slog.Error("failed to get phone number record", "error", err, "phone", phone)
		wp.failMessage(ctx, message.ID, "phone number not registered")
		return
	}

	companyID := pn.CompanyID

	if wp.companyRepo != nil && companyID != "" {
		ok, err := wp.companyRepo.TryIncrementQuota(ctx, companyID, 1)
		if err != nil {
			slog.Error("quota check failed", "error", err, "company_id", companyID)
			wp.failMessage(ctx, message.ID, "internal error")
			return
		}
		if !ok {
			slog.Warn("quota exceeded", "company_id", companyID, "message_id", message.ID)
			wp.failMessage(ctx, message.ID, "quota exceeded")
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
		return
	}

	if wp.msgRepo != nil {
		if err := wp.msgRepo.UpdateWAMessageID(ctx, message.ID, waMessageID); err != nil {
			slog.Error("failed to update WA message ID", "error", err, "message_id", message.ID)
		}
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

	slog.Info("message sent successfully", "message_id", message.ID, "wa_message_id", waMessageID)
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
