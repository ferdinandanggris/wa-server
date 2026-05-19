package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/wa-server/internal/models"
)

type PricingService interface {
	SyncFromMeta(ctx context.Context) (int, error)
	GetUsage(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error)
	GetSummary(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error)
}

type PricingHandler struct {
	svc PricingService
}

func NewPricingHandler(svc PricingService) *PricingHandler {
	return &PricingHandler{svc: svc}
}

func parseTimeParam(r *http.Request, name string, defaultDays int) time.Time {
	val := r.URL.Query().Get(name)
	if val == "" {
		if defaultDays > 0 {
			return time.Now().AddDate(0, 0, -defaultDays)
		}
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		t, err = time.Parse("2006-01-02", val)
		if err != nil {
			return time.Now().AddDate(0, 0, -defaultDays)
		}
	}
	return t
}

func (h *PricingHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		phone := r.URL.Query().Get("phone")
		if phone == "" {
			http.Error(w, "phone parameter is required", http.StatusBadRequest)
			return
		}
		start := parseTimeParam(r, "start", 30)
		end := parseTimeParam(r, "end", 0)
		if end.IsZero() {
			end = time.Now()
		}

		data, err := h.svc.GetUsage(r.Context(), phone, start, end)
		if err != nil {
			slog.Error("failed to get pricing usage", "error", err)
			http.Error(w, "failed to get pricing data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)

	case http.MethodPost:
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		synced, err := h.svc.SyncFromMeta(ctx)
		if err != nil {
			slog.Error("failed to sync pricing", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "pricing sync completed",
			"synced":  synced,
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *PricingHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := parseTimeParam(r, "start", 90)
	end := parseTimeParam(r, "end", 0)
	if end.IsZero() {
		end = time.Now()
	}

	data, err := h.svc.GetSummary(r.Context(), start, end)
	if err != nil {
		slog.Error("failed to get pricing summary", "error", err)
		http.Error(w, "failed to get pricing summary", http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = []models.PricingSummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *PricingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/pricing", h.handleRoot)
	mux.HandleFunc("/api/v1/pricing/summary", h.handleSummary)
}
