package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/wa-server/internal/models"
)

type PhoneNumberService interface {
	SyncFromMeta(ctx context.Context) (int, error)
	List(ctx context.Context) ([]models.PhoneNumber, error)
}

type PhoneNumberHandler struct {
	svc PhoneNumberService
}

func NewPhoneNumberHandler(svc PhoneNumberService) *PhoneNumberHandler {
	return &PhoneNumberHandler{svc: svc}
}

func (h *PhoneNumberHandler) list(w http.ResponseWriter, r *http.Request) {
	numbers, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to list phone numbers", "error", err)
		http.Error(w, "failed to list phone numbers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(numbers)
}

func (h *PhoneNumberHandler) sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	synced, err := h.svc.SyncFromMeta(ctx)
	if err != nil {
		slog.Error("failed to sync phone numbers", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "phone numbers sync completed",
		"synced":  synced,
	})
}

func (h *PhoneNumberHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/phone-numbers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.sync(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
