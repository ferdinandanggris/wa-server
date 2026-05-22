package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wa-server/internal/api/webhook"
	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/queue"
)

// OutboundHandler handles outgoing message API requests.
type OutboundHandler struct {
	msgRepo   models.MessageRepository
	convRepo  models.ConversationRepository
	queuePub  *queue.Publisher
	companyID string
	wsHub     *webhook.WebSocketHub
}

// NewOutboundHandler creates a new OutboundHandler.
func NewOutboundHandler(msgRepo models.MessageRepository, convRepo models.ConversationRepository, queuePub *queue.Publisher, companyID string, wsHub *webhook.WebSocketHub) *OutboundHandler {
	return &OutboundHandler{
		msgRepo:   msgRepo,
		convRepo:  convRepo,
		queuePub:  queuePub,
		companyID: companyID,
		wsHub:     wsHub,
	}
}

// SendMessageRequest is the JSON body for sending a message.
type SendMessageRequest struct {
	ConversationID string            `json:"conversation_id"`
	To             string            `json:"to"`
	MessageType    string            `json:"message_type"`
	Content        string            `json:"content"`
	SenderName     string            `json:"sender_name,omitempty"`
	MediaURL       string            `json:"media_url,omitempty"`
	MediaID        string            `json:"media_id,omitempty"`
	FileName       string            `json:"file_name,omitempty"`
	TemplateID     string            `json:"template_id,omitempty"`
	TemplateParams map[string]string `json:"template_params,omitempty"`
	TemplateName   string            `json:"template_name,omitempty"`
	LanguageCode   string            `json:"language_code,omitempty"`
	BodyParams     []string          `json:"body_params,omitempty"`
	ButtonParams   []string          `json:"button_params,omitempty"`
	HeaderParams   []string          `json:"header_params,omitempty"`
	ReactionEmoji  string            `json:"reaction_emoji,omitempty"`
	ReactionToMsg  string            `json:"reaction_to_message_id,omitempty"`
	ContextMsgID   string            `json:"context_message_id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// SendMessageResponse is returned after queuing a message.
type SendMessageResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

func (h *OutboundHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if req.ConversationID == "" && req.To == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "conversation_id or to is required"})
		return
	}

	if req.IdempotencyKey != "" {
		existing, err := h.msgRepo.GetByIdempotencyKey(r.Context(), req.IdempotencyKey)
		if err == nil && existing != nil {
			slog.Info("idempotent request, returning existing message", "idempotency_key", req.IdempotencyKey, "message_id", existing.ID)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"ok": true, "data": SendMessageResponse{MessageID: existing.ID, Status: existing.Status},
			})
			return
		}
	}

	messageType := req.MessageType
	if messageType == "" {
		messageType = "text"
	}

	conversationID := req.ConversationID
	slog.Info("send message request", "conversation_id", conversationID, "message_type", req.MessageType)

	if conversationID == "" {
		slog.Warn("conversation_id not provided, need to implement contact/conversation lookup")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "conversation_id is required"})
		return
	}

	content := req.Content
	if messageType == "reaction" && req.ReactionEmoji != "" {
		content = req.ReactionEmoji
	}

	paramsJSON := ""
	if req.TemplateParams != nil {
		data, _ := json.Marshal(req.TemplateParams)
		paramsJSON = string(data)
	}

	// Build template info for template messages
	templateID := req.TemplateID
	if templateID == "" && req.TemplateName != "" {
		templateID = req.TemplateName
	}

	msg := &models.Message{
		ID:             "",
		ConversationID: conversationID,
		Direction:      string(models.MessageDirectionOutbound),
		MessageType:    messageType,
		Content:        content,
		MediaURL:       req.MediaURL,
		TemplateID:     templateID,
		TemplateParams: paramsJSON,
		Status:         string(models.MessageStatusPending),
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      time.Now().UTC(),
	}

	if err := h.msgRepo.Create(r.Context(), msg); err != nil {
		slog.Error("failed to create message", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to create message"})
		return
	}

	preview := content
	if preview == "" {
		if messageType == "image" {
			preview = "🖼 Image"
		} else if messageType == "document" {
			preview = "📄 Document"
		} else if messageType == "reaction" {
			preview = content
		}
	}
	if conv, err := h.convRepo.GetByID(r.Context(), conversationID); err == nil {
		conv.LastMessagePreview = preview
		conv.UpdatedAt = time.Now().UTC()
		if err := h.convRepo.Update(r.Context(), conv); err != nil {
			slog.Error("failed to update conversation preview", "error", err)
		}
		if h.wsHub != nil {
			h.wsHub.BroadcastToCompany(conv.CompanyID, webhook.WebSocketMessage{
				Type: "UpdateConversation",
				Payload: map[string]interface{}{
					"id":                  conv.ID,
					"last_message_preview": preview,
					"unread_count":        conv.UnreadCount,
					"status":              conv.Status,
					"updated_at":          time.Now().UTC(),
				},
			})
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

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok": true, "data": SendMessageResponse{MessageID: msg.ID, Status: "pending"},
	})
}

func (h *OutboundHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "conversation_id is required"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	cursorIDStr := r.URL.Query().Get("cursor_id")
	offset := 0
	if cursorIDStr != "" {
		if cid, err := strconv.Atoi(cursorIDStr); err == nil {
			offset = cid
		}
	}

	messages, err := h.msgRepo.GetByConversationID(r.Context(), conversationID, limit, offset)
	if err != nil {
		slog.Error("failed to get messages", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to get messages"})
		return
	}

	if messages == nil {
		messages = []models.Message{}
	}

	hasMore := len(messages) >= limit
	nextCursorID := offset + len(messages)
	nextCursorUpdatedAt := ""
	if hasMore && len(messages) > 0 {
		nextCursorUpdatedAt = messages[len(messages)-1].CreatedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": map[string]interface{}{
			"items":                messages,
			"limit":                limit,
			"has_more":             hasMore,
			"next_cursor_id":       nextCursorID,
			"next_cursor_updated_at": nextCursorUpdatedAt,
		},
	})
}

func (h *OutboundHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.SendMessage(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		}
	})

}
