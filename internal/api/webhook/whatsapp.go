package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wa-server/internal/config"
	"github.com/wa-server/internal/models"
)

// WhatsAppHandler processes incoming WhatsApp webhook events.
type WhatsAppHandler struct {
	cfg          *config.Config
	msgRepo      models.MessageRepository
	contactRepo  models.ContactRepository
	convRepo     models.ConversationRepository
	messageQueue MessagePublisher
	wsHub        *WebSocketHub
}

// MessagePublisher sends messages to the RabbitMQ exchange for async processing.
type MessagePublisher interface {
	PublishInbound(ctx context.Context, msg *models.Message) error
	PublishOutbound(ctx context.Context, msg *models.Message) error
}

// NewWhatsAppHandler creates a new WhatsAppHandler.
func NewWhatsAppHandler(
	cfg *config.Config,
	msgRepo models.MessageRepository,
	contactRepo models.ContactRepository,
	convRepo models.ConversationRepository,
	queue MessagePublisher,
	wsHub *WebSocketHub,
) *WhatsAppHandler {
	return &WhatsAppHandler{
		cfg:          cfg,
		msgRepo:      msgRepo,
		contactRepo:  contactRepo,
		convRepo:     convRepo,
		messageQueue: queue,
		wsHub:        wsHub,
	}
}

func (h *WhatsAppHandler) Verify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	slog.Info("webhook verify request", "token", token, "expected", h.cfg.WhatsApp.VerifyToken)

	if token != h.cfg.WhatsApp.VerifyToken {
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}

	if challenge == "" {
		challenge = "test_challenge_for_verification"
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(challenge)); err != nil {
		slog.Error("failed to write webhook challenge response", "error", err)
	}
}

func (h *WhatsAppHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload WhatsAppWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to parse webhook payload", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if len(payload.Entry) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Value == nil {
				continue
			}

			for _, msg := range change.Value.Messages {
				h.processMessage(r.Context(), change.Value.Metadata, msg)
			}

			for _, status := range change.Value.Statuses {
				h.processStatus(r.Context(), status)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

const defaultCompanyID = "00000000-0000-0000-0000-000000000001"

func (h *WhatsAppHandler) processMessage(ctx context.Context, metadata *WhatsAppMetadata, msg WhatsAppMessage) {
	phoneNumber := extractPhone(msg.From)
	waID := msg.From

	contact, err := h.contactRepo.GetByWAID(ctx, metadata.PhoneNumberID, waID)
	if err != nil {
		contact = &models.Contact{
			ID:          "",
			CompanyID:   defaultCompanyID,
			WAID:        waID,
			PhoneNumber: phoneNumber,
			Name:        msg.FromProfile.Name,
			IsBlocked:   false,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if err := h.contactRepo.Upsert(ctx, contact); err != nil {
			slog.Error("failed to create contact", "error", err, "wa_id", waID)
			return
		}
		slog.Info("created new contact", "contact_id", contact.ID, "wa_id", waID)
	}

	companyID, err := h.resolveCompanyID(ctx, metadata.PhoneNumberID)
	if err != nil {
		slog.Error("failed to resolve company", "error", err, "phone_number_id", metadata.PhoneNumberID)
		return
	}

	slog.Info("looking for conversation", "company_id", companyID, "contact_id", contact.ID)
	conv, err := h.convRepo.GetByContactID(ctx, companyID, contact.ID)
	if err != nil {
		conv = &models.Conversation{
			ID:                    "",
			CompanyID:             companyID,
			ContactID:             contact.ID,
			Status:                string(models.ConversationStatusOpen),
			Is24hWindowActive:     true,
			UnreadCount:           1,
			LastCustomerMessageAt: timePtr(time.Now().UTC()),
			StartedAt:             time.Now().UTC(),
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}
		if err := h.convRepo.Create(ctx, conv); err != nil {
			slog.Error("failed to create conversation", "error", err)
			return
		}
	} else {
		if err := h.convRepo.IncrementUnread(ctx, conv.ID); err != nil {
			slog.Error("failed to increment unread count", "error", err, "conversation_id", conv.ID)
		}
		conv.UnreadCount++
		now := time.Now().UTC()
		conv.LastCustomerMessageAt = &now
		conv.Is24hWindowActive = true
		if err := h.convRepo.Update(ctx, conv); err != nil {
			slog.Error("failed to update conversation", "error", err, "conversation_id", conv.ID)
		}
	}

	messageType := parseMessageType(msg.Type)
	content := extractMessageContent(msg)

	message := &models.Message{
		ID:             "",
		ConversationID: conv.ID,
		MessageID:      msg.ID,
		Direction:      string(models.MessageDirectionInbound),
		MessageType:    string(messageType),
		Content:        content,
		Status:         string(models.MessageStatusPending),
		CreatedAt:      time.Now().UTC(),
	}

	if msg.Image != nil {
		message.MediaURL = msg.Image.URL
	}
	if msg.Video != nil {
		message.MediaURL = msg.Video.URL
	}
	if msg.Document != nil {
		message.MediaURL = msg.Document.URL
	}
	if msg.Audio != nil {
		message.MediaURL = msg.Audio.ID
	}
	if msg.Interactive != nil {
		message.Content = msg.Interactive.ButtonReply.Title
	}

	if err := h.msgRepo.Create(ctx, message); err != nil {
		slog.Error("failed to save message", "error", err)
		return
	}

	if h.messageQueue != nil {
		if err := h.messageQueue.PublishInbound(ctx, message); err != nil {
			slog.Error("failed to publish message to queue", "error", err)
		}
	}

	if h.wsHub != nil {
		h.wsHub.BroadcastToCompany(companyID, WebSocketMessage{
			Type:    "new_message",
			Payload: message,
		})
	}

	slog.Info("processed inbound message", "message_id", message.ID, "from", phoneNumber)
}

func (h *WhatsAppHandler) processStatus(ctx context.Context, status WhatsAppStatus) {
	msg, err := h.msgRepo.GetByMessageID(ctx, status.ID)
	if err != nil {
		slog.Error("failed to find message by message_id", "error", err, "message_id", status.ID, "new_status", status.Status)
		return
	}

	oldStatus := msg.Status
	now := time.Now().UTC()

	switch status.Status {
	case "sent":
		if err := h.msgRepo.UpdateDeliveryStatus(ctx, msg.ID, "sent", now); err != nil {
			slog.Error("failed to update sent status", "error", err)
			return
		}
	case "delivered":
		if err := h.msgRepo.UpdateDeliveryStatus(ctx, msg.ID, "delivered", now); err != nil {
			slog.Error("failed to update delivered status", "error", err)
			return
		}
	case "read":
		if err := h.msgRepo.UpdateDeliveryStatus(ctx, msg.ID, "read", now); err != nil {
			slog.Error("failed to update read status", "error", err)
			return
		}
	case "failed":
		errMsg := ""
		if len(status.Errors) > 0 {
			errMsg = status.Errors[0].Message
		}
		if err := h.msgRepo.SetFailed(ctx, msg.ID, errMsg); err != nil {
			slog.Error("failed to update failed status", "error", err)
			return
		}
	}

	if h.wsHub != nil {
		conv, err := h.convRepo.GetByID(ctx, msg.ConversationID)
		if err == nil {
			h.wsHub.BroadcastToCompany(conv.CompanyID, WebSocketMessage{
				Type:    "message_status",
				Payload: msg,
			})
		}
	}

	slog.Info("updated message status", "message_id", msg.ID, "old_status", oldStatus, "new_status", status.Status)
}

func (h *WhatsAppHandler) resolveCompanyID(ctx context.Context, phoneNumberID string) (string, error) {
	return defaultCompanyID, nil
}

func extractPhone(waID string) string {
	if strings.HasPrefix(waID, "=") {
		return strings.Trim(waID, "=")
	}
	if strings.HasPrefix(waID, "whatsapp:") {
		return strings.TrimPrefix(waID, "whatsapp:")
	}
	return waID
}

func parseMessageType(msgType string) models.MessageType {
	switch msgType {
	case "text":
		return models.MessageTypeText
	case "image":
		return models.MessageTypeImage
	case "video":
		return models.MessageTypeVideo
	case "document":
		return models.MessageTypeDocument
	case "audio":
		return models.MessageTypeAudio
	case "sticker":
		return models.MessageTypeSticker
	default:
		return models.MessageTypeText
	}
}

func extractMessageContent(msg WhatsAppMessage) string {
	if msg.Text != nil {
		return msg.Text.Body
	}
	if msg.Interactive != nil {
		if msg.Interactive.ButtonReply != nil {
			return msg.Interactive.ButtonReply.Title
		}
		if msg.Interactive.ListReply != nil {
			return msg.Interactive.ListReply.Title
		}
	}
	return ""
}

func timePtr(t time.Time) *time.Time {
	return &t
}

type WhatsAppWebhookPayload struct {
	Object string          `json:"object"`
	Entry  []WhatsAppEntry `json:"entry"`
}

type WhatsAppEntry struct {
	ID      string           `json:"id"`
	Time    int64            `json:"time"`
	Changes []WhatsAppChange `json:"changes"`
}

type WhatsAppChange struct {
	Value *WhatsAppValue `json:"value"`
	Field string         `json:"field"`
}

type WhatsAppValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         *WhatsAppMetadata `json:"metadata"`
	Messages         []WhatsAppMessage `json:"messages"`
	Statuses         []WhatsAppStatus  `json:"statuses"`
}

type WhatsAppMetadata struct {
	PhoneNumberID string `json:"phone_number_id"`
	DisplayName   string `json:"display_name"`
}

type WhatsAppMessage struct {
	ID          string               `json:"id"`
	From        string               `json:"from"`
	FromProfile WhatsAppProfile      `json:"from_profile"`
	Type        string               `json:"type"`
	Timestamp   string               `json:"timestamp"`
	Text        *WhatsAppText        `json:"text,omitempty"`
	Image       *WhatsAppMedia       `json:"image,omitempty"`
	Video       *WhatsAppMedia       `json:"video,omitempty"`
	Document    *WhatsAppMedia       `json:"document,omitempty"`
	Audio       *WhatsAppAudio       `json:"audio,omitempty"`
	Sticker     *WhatsAppMedia       `json:"sticker,omitempty"`
	Interactive *WhatsAppInteractive `json:"interactive,omitempty"`
	Context     *WhatsAppContext     `json:"context,omitempty"`
}

type WhatsAppProfile struct {
	Name string `json:"name"`
}

type WhatsAppText struct {
	Body string `json:"body"`
}

type WhatsAppMedia struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
}

type WhatsAppAudio struct {
	ID    string `json:"id"`
	Voice bool   `json:"voice"`
}

type WhatsAppInteractive struct {
	Type        string               `json:"type"`
	ButtonReply *WhatsAppButtonReply `json:"button_reply,omitempty"`
	ListReply   *WhatsAppListReply   `json:"list_reply,omitempty"`
}

type WhatsAppButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type WhatsAppListReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type WhatsAppContext struct {
	MessageID string `json:"message_id"`
}

type WhatsAppStatus struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	Timestamp string          `json:"timestamp"`
	Recipient string          `json:"recipient"`
	Errors    []WhatsAppError `json:"errors,omitempty"`
}

type WhatsAppError struct {
	Code      int                `json:"code"`
	Title     string             `json:"title"`
	Message   string             `json:"message"`
	ErrorData *WhatsAppErrorData `json:"error_data,omitempty"`
}

type WhatsAppErrorData struct {
	Details string `json:"details"`
}
