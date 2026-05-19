package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wa-server/internal/models"
)

type mockPricingSvc struct {
	syncFunc   func(ctx context.Context) (int, error)
	getUsage   func(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error)
	getSummary func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error)
}

func (m *mockPricingSvc) SyncFromMeta(ctx context.Context) (int, error) {
	return m.syncFunc(ctx)
}
func (m *mockPricingSvc) GetUsage(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
	return m.getUsage(ctx, phone, start, end)
}
func (m *mockPricingSvc) GetSummary(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
	return m.getSummary(ctx, start, end)
}

func TestPricingHandler_Sync(t *testing.T) {
	svc := &mockPricingSvc{
		syncFunc: func(ctx context.Context) (int, error) {
			return 3, nil
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pricing", nil)
	w := httptest.NewRecorder()
	h.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()

	if body["synced"] != 3.0 {
		t.Fatalf("synced = %v, want 3", body["synced"])
	}
}

func TestPricingHandler_Sync_Error(t *testing.T) {
	svc := &mockPricingSvc{
		syncFunc: func(ctx context.Context) (int, error) {
			return 0, errors.New("meta error")
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pricing", nil)
	w := httptest.NewRecorder()
	h.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPricingHandler_GetUsage(t *testing.T) {
	svc := &mockPricingSvc{
		getUsage: func(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
			return []models.WabaPricing{
				{PhoneNumber: phone, PricingCategory: "marketing", Volume: 10, Cost: 0.85},
			}, nil
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing?phone=%2B62811", nil)
	w := httptest.NewRecorder()
	h.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var data []models.WabaPricing
	json.NewDecoder(resp.Body).Decode(&data)
	resp.Body.Close()

	if len(data) != 1 || data[0].PhoneNumber != "+62811" {
		t.Fatalf("got %+v", data)
	}
}

func TestPricingHandler_GetUsage_MissingPhone(t *testing.T) {
	h := NewPricingHandler(&mockPricingSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing", nil)
	w := httptest.NewRecorder()
	h.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPricingHandler_GetUsage_Error(t *testing.T) {
	svc := &mockPricingSvc{
		getUsage: func(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing?phone=%2B62811", nil)
	w := httptest.NewRecorder()
	h.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPricingHandler_Summary(t *testing.T) {
	svc := &mockPricingSvc{
		getSummary: func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
			return []models.PricingSummary{
				{Month: "2026-05", PricingCategory: "marketing", TotalVolume: 100, TotalCost: 8.50},
			}, nil
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing/summary", nil)
	w := httptest.NewRecorder()
	h.handleSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var data []models.PricingSummary
	json.NewDecoder(resp.Body).Decode(&data)
	resp.Body.Close()

	if len(data) != 1 || data[0].PricingCategory != "marketing" {
		t.Fatalf("got %+v", data)
	}
}

func TestPricingHandler_Summary_Empty(t *testing.T) {
	svc := &mockPricingSvc{
		getSummary: func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
			return nil, nil
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing/summary", nil)
	w := httptest.NewRecorder()
	h.handleSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var data []models.PricingSummary
	json.NewDecoder(resp.Body).Decode(&data)
	resp.Body.Close()

	if data == nil || len(data) != 0 {
		t.Fatal("expected empty array, not null")
	}
}

func TestPricingHandler_Summary_Error(t *testing.T) {
	svc := &mockPricingSvc{
		getSummary: func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewPricingHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pricing/summary", nil)
	w := httptest.NewRecorder()
	h.handleSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPricingHandler_Summary_MethodNotAllowed(t *testing.T) {
	h := NewPricingHandler(&mockPricingSvc{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pricing/summary", nil)
	w := httptest.NewRecorder()
	h.handleSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPricingHandler_Routes(t *testing.T) {
	svc := &mockPricingSvc{
		getUsage: func(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
			return []models.WabaPricing{{PhoneNumber: phone}}, nil
		},
		getSummary: func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
			return []models.PricingSummary{{Month: "2026-05", PricingCategory: "marketing", TotalVolume: 10, TotalCost: 0.85}}, nil
		},
		syncFunc: func(ctx context.Context) (int, error) {
			return 2, nil
		},
	}

	h := NewPricingHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("sync", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/pricing", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("usage", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/pricing?phone=%2B62811")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("summary", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/pricing/summary")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("404-unknown", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/pricing/unknown")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
	})
}
