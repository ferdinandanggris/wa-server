package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wa-server/internal/api/webhook"
	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/phone"
	"github.com/wa-server/internal/repository"
)

type ConversationRepo interface {
	ListWithCursor(ctx context.Context, cursorID string, cursorUpdatedAt time.Time, limit int, search, filter string) ([]repository.ConversationRow, error)
	ListByPhoneNumberWithCursor(ctx context.Context, phoneNumber, cursorID string, cursorUpdatedAt time.Time, limit int, search, filter string) ([]repository.ConversationRow, error)
	GetUnreadSummaryByPhoneNumber(ctx context.Context) ([]repository.PhoneSummaryRow, error)
	GetByID(ctx context.Context, id string) (*models.Conversation, error)
	GetByPhoneNumberAndContact(ctx context.Context, phoneNumber, contactID string) (*models.Conversation, error)
	Create(ctx context.Context, conv *models.Conversation) error
	ResetUnread(ctx context.Context, id string) error
	Update(ctx context.Context, conv *models.Conversation) error
}

type ContactRepoForConversation interface {
	GetByWAID(ctx context.Context, companyID, waID string) (*models.Contact, error)
	Create(ctx context.Context, contact *models.Contact) error
	Upsert(ctx context.Context, contact *models.Contact) error
}

type ConversationHandler struct {
	convRepo    ConversationRepo
	contactRepo ContactRepoForConversation
	msgRepo     models.MessageRepository
	wsHub       *webhook.WebSocketHub
	wabaID      string
}

func NewConversationHandler(convRepo ConversationRepo, contactRepo ContactRepoForConversation, msgRepo models.MessageRepository, wsHub *webhook.WebSocketHub, wabaID string) *ConversationHandler {
	return &ConversationHandler{
		convRepo:    convRepo,
		contactRepo: contactRepo,
		msgRepo:     msgRepo,
		wsHub:       wsHub,
		wabaID:      wabaID,
	}
}

type conversationResponse struct {
	ID                  string  `json:"id"`
	WaChannelID         string  `json:"wa_channel_id"`
	WabaID              string  `json:"waba_id"`
	CustomerWAID        string  `json:"customer_wa_id"`
	CustomerName        string  `json:"customer_name"`
	DisplayNumber       string  `json:"display_number"`
	VerifiedName        string  `json:"verified_name"`
	LastMessagePreview  string  `json:"last_message_preview"`
	UnreadCount         int     `json:"unread_count"`
	Status              string  `json:"status"`
	IsTemplateRequired  bool    `json:"is_template_required"`
	UpdatedAt           string  `json:"updated_at"`
	LastMessageTimestamp *int64 `json:"last_message_timestamp,omitempty"`
}

func convToResponse(row repository.ConversationRow, wabaID string) conversationResponse {
	ts := func(t time.Time) *int64 {
		if t.IsZero() { return nil }
		v := t.Unix()
		return &v
	}

	lastTs := ts(row.UpdatedAt)
	if row.LastCustomerMessageAt != nil && !row.LastCustomerMessageAt.IsZero() {
		lastTs = ts(*row.LastCustomerMessageAt)
	}

	return conversationResponse{
		ID:                  row.ID,
		WaChannelID:         row.WaChannelID,
		VerifiedName:        row.VerifiedName,
		WabaID:              wabaID,
		CustomerWAID:        row.CustomerWAID,
		CustomerName:        row.CustomerName,
		DisplayNumber:       row.PhoneNumber,
		LastMessagePreview:  row.LastMessagePreview,
		UnreadCount:         row.UnreadCount,
		Status:              row.Status,
		IsTemplateRequired:  !row.Is24hWindowActive,
		UpdatedAt:           row.UpdatedAt.Format(time.RFC3339),
		LastMessageTimestamp: lastTs,
	}
}

func (h *ConversationHandler) listConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	search := q.Get("search")
	filter := q.Get("filter")
	if filter == "" {
		filter = "all"
	}
	phoneNumber := q.Get("phone_number")
	cursorID := q.Get("cursor_id")
	cursorUpdatedAtStr := q.Get("cursor_updated_at")

	var cursorUpdatedAt time.Time
	if cursorUpdatedAtStr != "" {
		cursorUpdatedAt, _ = time.Parse(time.RFC3339, cursorUpdatedAtStr)
	}

	var rows []repository.ConversationRow
	var err error

	if phoneNumber != "" {
		rows, err = h.convRepo.ListByPhoneNumberWithCursor(r.Context(), phoneNumber, cursorID, cursorUpdatedAt, limit, search, filter)
	} else {
		rows, err = h.convRepo.ListWithCursor(r.Context(), cursorID, cursorUpdatedAt, limit, search, filter)
	}

	if err != nil {
		slog.Error("failed to list conversations", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to list conversations"})
		return
	}

	convs := make([]conversationResponse, 0, len(rows))
	for _, row := range rows {
		convs = append(convs, convToResponse(row, h.wabaID))
	}

	hasMore := len(convs) >= limit
	nextCursorID := ""
	var nextCursorUpdatedAt string
	if hasMore && len(convs) > 0 {
		last := convs[len(convs)-1]
		nextCursorID = last.ID
		nextCursorUpdatedAt = last.UpdatedAt
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"items":                convs,
			"limit":                limit,
			"has_more":             hasMore,
			"next_cursor_id":       nextCursorID,
			"next_cursor_updated_at": nextCursorUpdatedAt,
		},
	})
}

type ensureConversationRequest struct {
	PhoneNumber  string `json:"phone_number"`
	CustomerWAID string `json:"customer_wa_id"`
	CustomerName string `json:"customer_name,omitempty"`
}

func (h *ConversationHandler) ensureConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	var req ensureConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if req.PhoneNumber == "" || req.CustomerWAID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "phone_number and customer_wa_id are required"})
		return
	}

	req.PhoneNumber = phone.Normalize(req.PhoneNumber)
	req.CustomerWAID = phone.Normalize(req.CustomerWAID)

	contact, err := h.contactRepo.GetByWAID(r.Context(), "", req.CustomerWAID)
	if err != nil {
		contact = &models.Contact{
			WAID:        req.CustomerWAID,
			PhoneNumber: req.CustomerWAID,
			Name:        req.CustomerName,
		}
		if err := h.contactRepo.Create(r.Context(), contact); err != nil {
			slog.Error("failed to create contact", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to create contact"})
			return
		}
	}

	existing, err := h.convRepo.GetByPhoneNumberAndContact(r.Context(), req.PhoneNumber, contact.ID)
	if err == nil && existing != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok": true, "data": map[string]interface{}{
				"id":           existing.ID,
				"phone_number": existing.PhoneNumber,
				"customer_wa_id": req.CustomerWAID,
				"customer_name": contact.Name,
				"status":       existing.Status,
			},
		})
		return
	}

	conv := &models.Conversation{
		PhoneNumber:     req.PhoneNumber,
		ContactID:       contact.ID,
		Status:          string(models.ConversationStatusOpen),
		Is24hWindowActive: true,
		LastMessagePreview: "",
	}
	if err := h.convRepo.Create(r.Context(), conv); err != nil {
		slog.Error("failed to create conversation", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to create conversation"})
		return
	}

	if h.wsHub != nil {
		h.wsHub.BroadcastToCompany(conv.CompanyID, webhook.WebSocketMessage{
			Type: "UpdateConversation",
			Payload: conv.ID,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok": true, "data": map[string]interface{}{
			"id": conv.ID,
			"phone_number": req.PhoneNumber,
			"customer_wa_id": req.CustomerWAID,
			"customer_name": contact.Name,
			"status": conv.Status,
		},
	})
}

func (h *ConversationHandler) markAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	convID := extractID(r.URL.Path, "/api/v1/conversations/")
	convID = strings.TrimSuffix(convID, "/read")
	if convID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "conversation id is required"})
		return
	}

	if err := h.convRepo.ResetUnread(r.Context(), convID); err != nil {
		slog.Error("failed to mark conversation as read", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to mark as read"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "data": map[string]string{"message": "conversation marked as read"}})
}

type renameConversationRequest struct {
	Name string `json:"name"`
}

func (h *ConversationHandler) renameConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	convID := extractID(r.URL.Path, "/api/v1/conversations/")
	convID = strings.TrimSuffix(convID, "/name")
	if convID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "conversation id is required"})
		return
	}

	var req renameConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	conv, err := h.convRepo.GetByID(r.Context(), convID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "conversation not found"})
		return
	}

	conv.Status = req.Name

	if err := h.convRepo.Update(r.Context(), conv); err != nil {
		slog.Error("failed to rename conversation", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to rename conversation"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "data": map[string]string{"message": "conversation renamed"}})
}

type typingRequest struct {
	Target     string `json:"target"`
	SenderName string `json:"sender_name"`
}

func (h *ConversationHandler) sendTyping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	convID := extractID(r.URL.Path, "/api/v1/conversations/")
	convID = strings.TrimSuffix(convID, "/typing")
	if convID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "conversation id is required"})
		return
	}

	var req typingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	if h.wsHub != nil {
		h.wsHub.BroadcastToCompany("", webhook.WebSocketMessage{
			Type: "AgentTyping",
			Payload: map[string]interface{}{
				"conversation_id": convID,
				"sender_name":     req.SenderName,
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "data": map[string]string{"message": "typing indicator sent"}})
}

func (h *ConversationHandler) handleConversationsPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/api/v1/conversations" || path == "/api/v1/conversations/":
		switch r.Method {
		case http.MethodGet:
			h.listConversations(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		}
	case strings.HasSuffix(path, "/ensure"):
		h.ensureConversation(w, r)
	case strings.HasSuffix(path, "/read"):
		h.markAsRead(w, r)
	case strings.HasSuffix(path, "/name"):
		h.renameConversation(w, r)
	case strings.HasSuffix(path, "/typing"):
		h.sendTyping(w, r)
	case strings.Contains(path, "/messages"):
		h.getConversationMessages(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "not found"})
	}
}

func (h *ConversationHandler) getConversationMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}
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
			"items":                  messages,
			"limit":                  limit,
			"has_more":               hasMore,
			"next_cursor_id":         nextCursorID,
			"next_cursor_updated_at": nextCursorUpdatedAt,
		},
	})
}

func (h *ConversationHandler) getPhoneSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	rows, err := h.convRepo.GetUnreadSummaryByPhoneNumber(r.Context())
	if err != nil {
		slog.Error("failed to get phone summary", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to get summary"})
		return
	}

	type summaryItem struct {
		ID              string `json:"id"`
		PhoneNumber     string `json:"phone_number"`
		DisplayName     string `json:"display_name"`
		VerifiedName    string `json:"verified_name"`
		IsActive        bool   `json:"is_active"`
		About           string `json:"about"`
		ProfilePicURL   string `json:"profile_picture_url"`
		UnreadCount     int    `json:"unread_count"`
	}

	items := make([]summaryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, summaryItem{
			ID:            row.ID,
			PhoneNumber:   row.PhoneNumber,
			DisplayName:   row.DisplayName,
			VerifiedName:  row.VerifiedName,
			IsActive:      row.IsActive,
			About:         row.About,
			ProfilePicURL: row.ProfilePicURL,
			UnreadCount:   row.UnreadCount,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": items,
	})
}

func (h *ConversationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/conversations", h.handleConversationsPath)
	mux.HandleFunc("/api/v1/conversations/", h.handleConversationsPath)
	mux.HandleFunc("/api/v1/phone-numbers/summary", h.getPhoneSummary)
}
