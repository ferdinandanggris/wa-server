package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/queue"
	"github.com/wa-server/internal/repository"
)

type OutboundHandler struct {
	msgRepo     models.MessageRepository
	queuePub    *queue.Publisher
	companyID   string
	waClient    WhatsAppClient
	contactRepo *repository.ContactRepository
}

type WhatsAppClient interface {
	SendMessage(ctx context.Context, to, messageType, content, mediaURL string) (string, error)
}

func NewOutboundHandler(msgRepo models.MessageRepository, queuePub *queue.Publisher, companyID string, waClient WhatsAppClient, contactRepo *repository.ContactRepository) *OutboundHandler {
	return &OutboundHandler{
		msgRepo:     msgRepo,
		queuePub:    queuePub,
		companyID:   companyID,
		waClient:    waClient,
		contactRepo: contactRepo,
	}
}

type SendMessageRequest struct {
	ConversationID string            `json:"conversation_id"`
	To             string            `json:"to"`
	MessageType    string            `json:"message_type"`
	Content        string            `json:"content"`
	MediaURL       string            `json:"media_url,omitempty"`
	TemplateID     string            `json:"template_id,omitempty"`
	TemplateParams map[string]string `json:"template_params,omitempty"`
}

type SendMessageResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

func (h *OutboundHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.ConversationID == "" && req.To == "" {
		http.Error(w, "conversation_id or to is required", http.StatusBadRequest)
		return
	}

	messageType := req.MessageType
	if messageType == "" {
		messageType = "text"
	}

	conversationID := req.ConversationID
	slog.Info("send message request", "conversation_id", conversationID, "content", req.Content, "message_type", req.MessageType)

	if conversationID == "" {
		// TODO: Create conversation from phone number
		slog.Warn("conversation_id not provided, need to implement contact/conversation lookup")
		http.Error(w, "conversation_id is required for now", http.StatusBadRequest)
		return
	}

	paramsJSON := ""
	if req.TemplateParams != nil {
		data, _ := json.Marshal(req.TemplateParams)
		paramsJSON = string(data)
	}

	msg := &models.Message{
		ID:             "",
		ConversationID: conversationID,
		Direction:      string(models.MessageDirectionOutbound),
		MessageType:    messageType,
		Content:        req.Content,
		MediaURL:       req.MediaURL,
		TemplateID:     req.TemplateID,
		TemplateParams: paramsJSON,
		Status:         string(models.MessageStatusPending),
		CreatedAt:      time.Now().UTC(),
	}

	if err := h.msgRepo.Create(r.Context(), msg); err != nil {
		slog.Error("failed to create message", "error", err, "conversation_id", msg.ConversationID, "message_id", msg.MessageID, "content", msg.Content)
		http.Error(w, "failed to create message", http.StatusInternalServerError)
		return
	}

	// Direct WhatsApp call for immediate send (bypass worker for testing)
	if h.waClient != nil && h.contactRepo != nil {
		phone, err := h.contactRepo.GetPhoneByConversationID(r.Context(), conversationID)
		if err != nil {
			slog.Warn("failed to get phone for direct send", "error", err)
		} else {
			slog.Info("sending directly to WhatsApp", "phone", phone)
			waMsgID, sendErr := h.waClient.SendMessage(r.Context(), phone, messageType, req.Content, req.MediaURL)
			if sendErr != nil {
				slog.Error("direct WhatsApp send failed", "error", sendErr)
			} else {
				slog.Info("direct WhatsApp send success", "wa_message_id", waMsgID)
				msg.Status = string(models.MessageStatusSent)
				msg.MessageID = waMsgID
				if err := h.msgRepo.UpdateStatus(r.Context(), msg.ID, msg.Status); err != nil {
					slog.Error("failed to update message status", "error", err)
				}
				if err := h.msgRepo.UpdateWAMessageID(r.Context(), msg.ID, waMsgID); err != nil {
					slog.Error("failed to update WA message ID", "error", err)
				}
			}
		}
	}

	if h.queuePub != nil {
		slog.Info("publishing message to queue", "message_id", msg.ID)
		if err := h.queuePub.PublishOutbound(r.Context(), msg); err != nil {
			slog.Error("failed to publish message to queue", "error", err)
		}
	} else {
		slog.Warn("queuePub is nil, message not queued")
	}

	resp := SendMessageResponse{
		MessageID: msg.ID,
		Status:    "pending",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func (h *OutboundHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	// Extract conversation ID from path: /api/v1/conversations/{id}/messages
	path := r.URL.Path
	conversationID := ""
	if strings.HasPrefix(path, "/api/v1/conversations/") {
		rest := strings.TrimPrefix(path, "/api/v1/conversations/")
		if idx := strings.Index(rest, "/messages"); idx > 0 {
			conversationID = rest[:idx]
		}
	}

	if conversationID == "" {
		conversationID = r.URL.Query().Get("conversation_id")
	}

	if conversationID == "" {
		http.Error(w, "conversation_id is required", http.StatusBadRequest)
		return
	}

	messages, err := h.msgRepo.GetByConversationID(r.Context(), conversationID, 100, 0)
	if err != nil {
		slog.Error("failed to get messages", "error", err)
		http.Error(w, "failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		slog.Error("failed to encode messages", "error", err)
	}
}

func (h *OutboundHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.SendMessage(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/conversations/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetMessages(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
