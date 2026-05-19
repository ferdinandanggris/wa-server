package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/wa-server/internal/models"
)

// BillingService defines billing operations needed by the HTTP handler.
type BillingService interface {
	GetUsage(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error)
	GetQuota(ctx context.Context, companyID string) (*models.Company, error)
	SyncCostsFromMeta(ctx context.Context, start, end time.Time) (int, error)
	GetCostSummary(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error)
}

// BillingHandler serves REST endpoints for billing.
type BillingHandler struct {
	svc BillingService
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(svc BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

func (h *BillingHandler) getUsage(w http.ResponseWriter, r *http.Request) {
	companyID := r.URL.Query().Get("company_id")
	if companyID == "" {
		http.Error(w, "company_id is required", http.StatusBadRequest)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	logs, err := h.svc.GetUsage(r.Context(), companyID, start, end)
	if err != nil {
		slog.Error("failed to get usage", "error", err)
		http.Error(w, "failed to get usage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *BillingHandler) getQuota(w http.ResponseWriter, r *http.Request) {
	companyID := r.URL.Query().Get("company_id")
	if companyID == "" {
		http.Error(w, "company_id is required", http.StatusBadRequest)
		return
	}

	company, err := h.svc.GetQuota(r.Context(), companyID)
	if err != nil {
		slog.Error("failed to get quota", "error", err)
		http.Error(w, "company not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"company_id":  company.ID,
		"quota_limit": company.QuotaLimit,
		"quota_used":  company.QuotaUsed,
		"remaining":   company.QuotaLimit - company.QuotaUsed,
	})
}

func (h *BillingHandler) syncCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	start := time.Now().AddDate(0, -7, 0)
	end := time.Now()

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	updated, err := h.svc.SyncCostsFromMeta(r.Context(), start, end)
	if err != nil {
		slog.Error("failed to sync costs", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "cost sync completed",
		"updated": updated,
	})
}

func (h *BillingHandler) getCostSummary(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	start := time.Now().AddDate(0, -7, 0)
	end := time.Now()

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	summary, err := h.svc.GetCostSummary(r.Context(), start, end)
	if err != nil {
		slog.Error("failed to get cost summary", "error", err)
		http.Error(w, "failed to get cost summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (h *BillingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/billing/usage", h.getUsage)
	mux.HandleFunc("/api/v1/billing/quota", h.getQuota)
	mux.HandleFunc("/api/v1/billing/sync-costs", h.syncCosts)
	mux.HandleFunc("/api/v1/billing/cost-summary", h.getCostSummary)
}
