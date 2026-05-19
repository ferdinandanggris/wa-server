package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wa-server/internal/models"
)

type mockBillingSvc struct {
	getUsageFunc       func(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error)
	getQuotaFunc       func(ctx context.Context, companyID string) (*models.Company, error)
	syncCostsFunc      func(ctx context.Context, start, end time.Time) (int, error)
	getCostSummaryFunc func(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error)
}

func (m *mockBillingSvc) GetUsage(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	return m.getUsageFunc(ctx, companyID, startDate, endDate)
}
func (m *mockBillingSvc) GetQuota(ctx context.Context, companyID string) (*models.Company, error) {
	return m.getQuotaFunc(ctx, companyID)
}
func (m *mockBillingSvc) SyncCostsFromMeta(ctx context.Context, start, end time.Time) (int, error) {
	return m.syncCostsFunc(ctx, start, end)
}
func (m *mockBillingSvc) GetCostSummary(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error) {
	return m.getCostSummaryFunc(ctx, start, end)
}

func TestBillingHandler_GetQuota(t *testing.T) {
	svc := &mockBillingSvc{
		getQuotaFunc: func(ctx context.Context, companyID string) (*models.Company, error) {
			return &models.Company{ID: companyID, QuotaLimit: 1000, QuotaUsed: 50}, nil
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/quota?company_id=c1", nil)
	w := httptest.NewRecorder()
	h.getQuota(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()

	if body["company_id"] != "c1" {
		t.Fatalf("company_id = %v, want c1", body["company_id"])
	}
	if body["remaining"] != 950.0 {
		t.Fatalf("remaining = %v, want 950", body["remaining"])
	}
}

func TestBillingHandler_GetQuota_MissingCompanyID(t *testing.T) {
	h := NewBillingHandler(&mockBillingSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/quota", nil)
	w := httptest.NewRecorder()
	h.getQuota(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestBillingHandler_GetQuota_NotFound(t *testing.T) {
	svc := &mockBillingSvc{
		getQuotaFunc: func(ctx context.Context, companyID string) (*models.Company, error) {
			return nil, context.DeadlineExceeded
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/quota?company_id=nonexistent", nil)
	w := httptest.NewRecorder()
	h.getQuota(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestBillingHandler_GetUsage(t *testing.T) {
	svc := &mockBillingSvc{
		getUsageFunc: func(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			return []models.BillingLog{
				{ID: "bl1", CompanyID: companyID, PhoneNumber: "+62811", TemplateCost: 0.05},
			}, nil
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/usage?company_id=c1", nil)
	w := httptest.NewRecorder()
	h.getUsage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var logs []models.BillingLog
	json.NewDecoder(resp.Body).Decode(&logs)
	resp.Body.Close()

	if len(logs) != 1 || logs[0].ID != "bl1" {
		t.Fatalf("got %+v", logs)
	}
}

func TestBillingHandler_GetUsage_MissingCompanyID(t *testing.T) {
	h := NewBillingHandler(&mockBillingSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/usage", nil)
	w := httptest.NewRecorder()
	h.getUsage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestBillingHandler_SyncCosts(t *testing.T) {
	svc := &mockBillingSvc{
		syncCostsFunc: func(ctx context.Context, start, end time.Time) (int, error) {
			return 5, nil
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/sync-costs", nil)
	w := httptest.NewRecorder()
	h.syncCosts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()

	if body["updated"] != 5.0 {
		t.Fatalf("updated = %v, want 5", body["updated"])
	}
}

func TestBillingHandler_SyncCosts_MethodNotAllowed(t *testing.T) {
	h := NewBillingHandler(&mockBillingSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/sync-costs", nil)
	w := httptest.NewRecorder()
	h.syncCosts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestBillingHandler_SyncCosts_Error(t *testing.T) {
	svc := &mockBillingSvc{
		syncCostsFunc: func(ctx context.Context, start, end time.Time) (int, error) {
			return 0, context.DeadlineExceeded
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/sync-costs", nil)
	w := httptest.NewRecorder()
	h.syncCosts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestBillingHandler_GetCostSummary(t *testing.T) {
	svc := &mockBillingSvc{
		getCostSummaryFunc: func(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error) {
			return []models.BillingCostSummary{
				{PhoneNumber: "+62811", TotalMessages: 10, TotalCost: 0.50},
			}, nil
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/cost-summary", nil)
	w := httptest.NewRecorder()
	h.getCostSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var summary []models.BillingCostSummary
	json.NewDecoder(resp.Body).Decode(&summary)
	resp.Body.Close()

	if len(summary) != 1 || summary[0].TotalCost != 0.50 {
		t.Fatalf("got %+v", summary)
	}
}

func TestBillingHandler_RegisterRoutes(t *testing.T) {
	svc := &mockBillingSvc{
		getQuotaFunc: func(ctx context.Context, companyID string) (*models.Company, error) {
			return &models.Company{ID: companyID, QuotaLimit: 1000, QuotaUsed: 50}, nil
		},
		getUsageFunc: func(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			return []models.BillingLog{{ID: "bl1"}}, nil
		},
		getCostSummaryFunc: func(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error) {
			return []models.BillingCostSummary{{PhoneNumber: "+62811", TotalCost: 0.50}}, nil
		},
		syncCostsFunc: func(ctx context.Context, start, end time.Time) (int, error) {
			return 3, nil
		},
	}

	h := NewBillingHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("quota", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/billing/quota?company_id=c1")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("usage", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/billing/usage?company_id=c1")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("cost-summary", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/billing/cost-summary")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("sync-costs", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/billing/sync-costs", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("404-unknown-path", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/billing/unknown")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
	})
}

func TestBillingHandler_GetCostSummary_Error(t *testing.T) {
	svc := &mockBillingSvc{
		getCostSummaryFunc: func(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error) {
			return nil, context.DeadlineExceeded
		},
	}
	h := NewBillingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/cost-summary", nil)
	w := httptest.NewRecorder()
	h.getCostSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}
