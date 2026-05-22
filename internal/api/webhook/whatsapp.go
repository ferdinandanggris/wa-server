package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wa-server/internal/config"
	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/phone"
)

type WhatsAppHandler struct {
	cfg             *config.Config
	msgRepo         models.MessageRepository
	contactRepo     models.ContactRepository
	convRepo        models.ConversationRepository
	tmplRepo        models.TemplateRepository
	phoneNumberRepo models.PhoneNumberRepository
	messageQueue    MessagePublisher
	wsHub           *WebSocketHub
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
	tmplRepo models.TemplateRepository,
	phoneNumberRepo models.PhoneNumberRepository,
	queue MessagePublisher,
	wsHub *WebSocketHub,
) *WhatsAppHandler {
	return &WhatsAppHandler{
		cfg:             cfg,
		msgRepo:         msgRepo,
		contactRepo:     contactRepo,
		convRepo:        convRepo,
		tmplRepo:        tmplRepo,
		phoneNumberRepo: phoneNumberRepo,
		messageQueue:    queue,
		wsHub:           wsHub,
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

	slog.Info("incoming webhook",
		"entry_count", len(payload.Entry),
		"body", string(body),
	)

	if len(payload.Entry) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			switch change.Field {
			case "message_template_status_update":
				h.processTemplateStatusUpdate(r.Context(), change.Raw)
			case "messages":
				if change.Value == nil {
					continue
				}
				slog.Info("webhook messages change",
					"meta_phone_number_id", change.Value.Metadata.PhoneNumberID,
					"display_name", change.Value.Metadata.DisplayName,
					"message_count", len(change.Value.Messages),
					"status_count", len(change.Value.Statuses),
					"contact_count", len(change.Value.Contacts),
				)
				contactNames := map[string]string{}
				for _, c := range change.Value.Contacts {
					contactNames[c.WAID] = c.Profile.Name
				}
				for _, msg := range change.Value.Messages {
					slog.Info("webhook message detail",
						"from", msg.From,
						"msg_id", msg.ID,
						"msg_type", msg.Type,
						"timestamp", msg.Timestamp,
					)
					h.processMessage(r.Context(), change.Value.Metadata, msg, contactNames[msg.From])
				}
				for _, status := range change.Value.Statuses {
					slog.Info("webhook status detail",
						"status_id", status.ID,
						"status", status.Status,
						"recipient", status.Recipient,
						"timestamp", status.Timestamp,
						"error_count", len(status.Errors),
					)
					h.processStatus(r.Context(), status)
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

const defaultCompanyID = "00000000-0000-0000-0000-000000000001"

func (h *WhatsAppHandler) processMessage(ctx context.Context, metadata *WhatsAppMetadata, msg WhatsAppMessage, contactName string) {
	phoneNumber := phone.Normalize(extractPhone(msg.From))
	waID := msg.From

	contact := &models.Contact{
		CompanyID:   defaultCompanyID,
		WAID:        waID,
		PhoneNumber: phoneNumber,
		Name:        contactName,
		IsBlocked:   false,
		UpdatedAt:   time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}
	existing, err := h.contactRepo.GetByWAID(ctx, defaultCompanyID, waID)
	if err == nil {
		contact.ID = existing.ID
		contact.CreatedAt = existing.CreatedAt
		contact.Name = existing.Name
	}
	if err := h.contactRepo.Upsert(ctx, contact); err != nil {
		slog.Error("failed to upsert contact", "error", err, "wa_id", waID)
		return
	}

	companyID, err := h.resolveCompanyID(ctx, metadata.PhoneNumberID)
	if err != nil {
		slog.Error("failed to resolve company", "error", err, "phone_number_id", metadata.PhoneNumberID)
		return
	}

	var localPhoneNumberID string
	pn, err := h.phoneNumberRepo.GetByMetaID(ctx, metadata.PhoneNumberID)
	if err == nil {
		localPhoneNumberID = pn.ID
	}

	slog.Info("looking for conversation", "phone_number", phoneNumber, "contact_id", contact.ID)
	content := extractMessageContent(msg)
	conv, err := h.convRepo.GetByPhoneNumberAndContact(ctx, phoneNumber, contact.ID)
	if err != nil {
		conv = &models.Conversation{
			ID:                    "",
			CompanyID:             companyID,
			ContactID:             contact.ID,
			PhoneNumber:           phoneNumber,
			PhoneNumberID:         localPhoneNumberID,
			Status:                string(models.ConversationStatusOpen),
			Is24hWindowActive:     true,
			UnreadCount:           1,
			LastMessagePreview:    content,
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
		conv.LastMessagePreview = content
		conv.Is24hWindowActive = true
		if err := h.convRepo.Update(ctx, conv); err != nil {
			slog.Error("failed to update conversation", "error", err, "conversation_id", conv.ID)
		}
	}

	existingMsg, _ := h.msgRepo.GetByMessageID(ctx, msg.ID)
	if existingMsg != nil {
		slog.Info("duplicate inbound message, skipping", "meta_message_id", msg.ID)
		return
	}

	messageType := parseMessageType(msg.Type)

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
			Type:    "ReceiveMessage",
			Payload: message,
		})
		h.wsHub.BroadcastToCompany(companyID, WebSocketMessage{
			Type:    "UpdateConversation",
			Payload: map[string]interface{}{
				"id":                  conv.ID,
				"last_message_preview": conv.LastMessagePreview,
				"unread_count":        conv.UnreadCount,
				"status":              conv.Status,
				"updated_at":          conv.UpdatedAt,
			},
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

	if !isValidStatusTransition(oldStatus, status.Status) {
		slog.Info("skipping backward status transition", "message_id", msg.ID, "old_status", oldStatus, "new_status", status.Status)
		return
	}

	now := time.Now().UTC()

	switch status.Status {
	case "sent":
		if err := h.msgRepo.UpdateDeliveryStatus(ctx, msg.ID, "sent", now); err != nil {
			slog.Error("failed to update sent status", "error", err)
			return
		}
		msg.Status = "sent"
		msg.SentAt = &now
	case "delivered":
		if err := h.msgRepo.UpdateDeliveryStatus(ctx, msg.ID, "delivered", now); err != nil {
			slog.Error("failed to update delivered status", "error", err)
			return
		}
		msg.Status = "delivered"
		msg.DeliveredAt = &now
	case "read":
		if err := h.msgRepo.UpdateDeliveryStatus(ctx, msg.ID, "read", now); err != nil {
			slog.Error("failed to update read status", "error", err)
			return
		}
		msg.Status = "read"
		msg.ReadAt = &now
	case "failed":
		errMsg := ""
		if len(status.Errors) > 0 {
			errMsg = status.Errors[0].Message
		}
		if err := h.msgRepo.SetFailed(ctx, msg.ID, errMsg); err != nil {
			slog.Error("failed to update failed status", "error", err)
			return
		}
		msg.Status = "failed"
		msg.ErrorMessage = errMsg
	}

	if h.wsHub != nil {
		conv, err := h.convRepo.GetByID(ctx, msg.ConversationID)
		if err == nil {
			slog.Info("broadcasting MessageStatusUpdated from webhook", "message_id", msg.ID, "company_id", conv.CompanyID, "status", status.Status)
			h.wsHub.BroadcastToCompany(conv.CompanyID, WebSocketMessage{
				Type:    "MessageStatusUpdated",
				Payload: msg,
			})
		} else {
			slog.Error("failed to get conversation for broadcast", "error", err, "conversation_id", msg.ConversationID)
		}
	} else {
		slog.Warn("wsHub is nil, skipping status broadcast from webhook", "message_id", msg.ID)
	}

	slog.Info("updated message status", "message_id", msg.ID, "old_status", oldStatus, "new_status", status.Status)
}

func (h *WhatsAppHandler) processTemplateStatusUpdate(ctx context.Context, raw map[string]interface{}) {
	if raw == nil {
		return
	}

	event, _ := raw["event"].(string)
	tmplName, _ := raw["message_template_name"].(string)
	tmplLang, _ := raw["message_template_language"].(string)

	slog.Info("template status update from webhook",
		"event", event,
		"template_name", tmplName,
		"language", tmplLang,
	)

	if tmplName == "" || tmplLang == "" {
		slog.Warn("incomplete template status update, skipping")
		return
	}

	tmpl, err := h.tmplRepo.GetByMetaNameAndLanguage(ctx, tmplName, tmplLang)
	if err != nil {
		slog.Warn("template not found from webhook update, skipping", "meta_name", tmplName, "language", tmplLang)
		return
	}

	if id, ok := raw["message_template_id"].(float64); ok {
		tmpl.WATemplateID = fmt.Sprintf("%.0f", id)
	}

	switch event {
	case "APPROVED":
		tmpl.MetaStatus = "APPROVED"
		tmpl.IsVerified = true
	case "REJECTED", "DISABLED", "PAUSED", "FLAGGED":
		tmpl.MetaStatus = event
		tmpl.IsVerified = false
	case "PENDING":
		tmpl.MetaStatus = "PENDING"
		tmpl.IsVerified = false
	default:
		tmpl.MetaStatus = event
	}

	if err := h.tmplRepo.Update(ctx, tmpl); err != nil {
		slog.Error("failed to update template from webhook", "error", err, "id", tmpl.ID)
		return
	}

	slog.Info("template status updated from webhook", "id", tmpl.ID, "status", tmpl.MetaStatus, "is_verified", tmpl.IsVerified)
}

func (h *WhatsAppHandler) resolveCompanyID(ctx context.Context, phoneNumberID string) (string, error) {
	return defaultCompanyID, nil
}

func isValidStatusTransition(oldStatus, newStatus string) bool {
	switch oldStatus {
	case "pending":
		return newStatus == "sent" || newStatus == "failed"
	case "sent":
		return newStatus == "delivered" || newStatus == "failed"
	case "delivered":
		return newStatus == "read"
	case "read", "failed":
		return false
	default:
		return true
	}
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
	Value *WhatsAppValue         `json:"value,omitempty"`
	Field string                 `json:"field"`
	Raw   map[string]interface{} `json:"-"`
}

func (c *WhatsAppChange) UnmarshalJSON(data []byte) error {
	type alias WhatsAppChange
	aux := &struct {
		Value json.RawMessage `json:"value"`
		*alias
	}{
		alias: (*alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if c.Field == "messages" && len(aux.Value) > 0 {
		var v WhatsAppValue
		if err := json.Unmarshal(aux.Value, &v); err != nil {
			return err
		}
		c.Value = &v
	} else if len(aux.Value) > 0 {
		var raw map[string]interface{}
		if err := json.Unmarshal(aux.Value, &raw); err != nil {
			return err
		}
		c.Raw = raw
	}
	return nil
}

type WhatsAppContact struct {
	Profile WhatsAppProfile `json:"profile"`
	WAID    string           `json:"wa_id"`
}

type WhatsAppValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         *WhatsAppMetadata `json:"metadata"`
	Messages         []WhatsAppMessage `json:"messages"`
	Statuses         []WhatsAppStatus  `json:"statuses"`
	Contacts         []WhatsAppContact `json:"contacts"`
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
