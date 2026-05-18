package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/wa-server/internal/config"
	"github.com/wa-server/internal/models"
)

type WhatsAppClient interface {
	SendMessage(ctx context.Context, phoneNumberID, to, messageType, content string, mediaURL string) (string, error)
	SendTemplateMessage(ctx context.Context, phoneNumberID, to, templateID string, params map[string]string) (string, error)
}

type WorkerPool struct {
	rmq       *RabbitMQ
	whatsapp  WhatsAppClient
	msgRepo   models.MessageRepository
	companyID string
	workers   int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewWorkerPool(rmq *RabbitMQ, whatsapp WhatsAppClient, msgRepo models.MessageRepository, workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		rmq:      rmq,
		whatsapp: whatsapp,
		msgRepo:  msgRepo,
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
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
	defer wp.wg.Done()

	msgs, err := wp.rmq.Consume(wp.ctx, QueueOutbound)
	if err != nil {
		slog.Error("worker failed to start consumer", "worker_id", id, "error", err)
		return
	}

	slog.Info("worker started", "worker_id", id)

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
			wp.processMessage(wp.ctx, msg.Body)
			msg.Ack(false)
		}
	}
}

func (wp *WorkerPool) processMessage(ctx context.Context, body []byte) {
	var message struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		MessageType    string `json:"message_type"`
		Content        string `json:"content"`
		MediaURL       string `json:"media_url"`
		TemplateID     string `json:"template_id"`
		TemplateParams string `json:"template_params"`
	}

	if err := json.Unmarshal(body, &message); err != nil {
		slog.Error("failed to unmarshal message", "error", err)
		return
	}

	slog.Info("processing outbound message", "message_id", message.ID)

	phoneNumberID := "default"   // TODO: Resolve from conversation/company
	to := message.ConversationID // TODO: Resolve to phone number

	var waMessageID string
	var err error

	if message.MessageType == "template" {
		params := parseTemplateParams(message.TemplateParams)
		waMessageID, err = wp.whatsapp.SendTemplateMessage(ctx, phoneNumberID, to, message.TemplateID, params)
	} else {
		waMessageID, err = wp.whatsapp.SendMessage(ctx, phoneNumberID, to, message.MessageType, message.Content, message.MediaURL)
	}

	if err != nil {
		slog.Error("failed to send message to WhatsApp", "error", err, "message_id", message.ID)
		wp.failMessage(ctx, message.ID, err.Error())
		return
	}

	if wp.msgRepo != nil {
		if err := wp.msgRepo.UpdateWAMessageID(ctx, message.ID, waMessageID); err != nil {
			slog.Error("failed to update WA message ID", "error", err, "message_id", message.ID)
		}
	}

	slog.Info("message sent successfully", "message_id", message.ID, "wa_message_id", waMessageID)
}

func (wp *WorkerPool) failMessage(ctx context.Context, messageID, errMsg string) {
	if wp.msgRepo != nil {
		if err := wp.msgRepo.UpdateStatus(ctx, messageID, string(models.MessageStatusFailed)); err != nil {
			slog.Error("failed to update message status", "error", err)
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

type Config struct {
	RabbitMQ *config.RabbitMQConfig
	WhatsApp *config.WhatsAppConfig
	Workers  int `envconfig:"WORKER_POOL_WORKERS" default:"5"`
}

type OutboundMessage struct {
	ID             string            `json:"id"`
	ConversationID string            `json:"conversation_id"`
	CompanyID      string            `json:"company_id"`
	To             string            `json:"to"`
	MessageType    string            `json:"message_type"`
	Content        string            `json:"content"`
	MediaURL       string            `json:"media_url,omitempty"`
	TemplateID     string            `json:"template_id,omitempty"`
	TemplateParams map[string]string `json:"template_params,omitempty"`
}
