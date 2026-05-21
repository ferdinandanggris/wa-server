package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wa-server/internal/models"
)

type mockPhoneNumberSvc struct {
	syncFunc   func(ctx context.Context) (int, error)
	listFunc   func(ctx context.Context) ([]models.PhoneNumber, error)
	assignFunc func(ctx context.Context, id, companyID string) (*models.PhoneNumber, error)
	getProfFunc func(ctx context.Context, id string) (*models.WhatsAppBusinessProfile, error)
	updProfFunc func(ctx context.Context, id string, profile *models.WhatsAppBusinessProfile) error
	updActFunc  func(ctx context.Context, id string, isActive bool) (*models.PhoneNumber, error)
}

func (m *mockPhoneNumberSvc) SyncFromMeta(ctx context.Context) (int, error) {
	return m.syncFunc(ctx)
}
func (m *mockPhoneNumberSvc) List(ctx context.Context) ([]models.PhoneNumber, error) {
	return m.listFunc(ctx)
}
func (m *mockPhoneNumberSvc) AssignToCompany(ctx context.Context, id, companyID string) (*models.PhoneNumber, error) {
	return m.assignFunc(ctx, id, companyID)
}
func (m *mockPhoneNumberSvc) GetProfile(ctx context.Context, id string) (*models.WhatsAppBusinessProfile, error) {
	return m.getProfFunc(ctx, id)
}
func (m *mockPhoneNumberSvc) UpdateProfile(ctx context.Context, id string, profile *models.WhatsAppBusinessProfile) error {
	return m.updProfFunc(ctx, id, profile)
}
func (m *mockPhoneNumberSvc) UpdateIsActive(ctx context.Context, id string, isActive bool) (*models.PhoneNumber, error) {
	if m.updActFunc != nil {
		return m.updActFunc(ctx, id, isActive)
	}
	return &models.PhoneNumber{}, nil
}

func TestPhoneNumberHandler_List(t *testing.T) {
	svc := &mockPhoneNumberSvc{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{
				{PhoneNumber: "+62811", PhoneNumberID: "pn1", IsActive: true},
			}, nil
		},
	}
	h := NewPhoneNumberHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()

	if body["ok"] != true {
		t.Fatalf("ok = %v, want true", body["ok"])
	}
}

func TestPhoneNumberHandler_List_Error(t *testing.T) {
	svc := &mockPhoneNumberSvc{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewPhoneNumberHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPhoneNumberHandler_Sync(t *testing.T) {
	svc := &mockPhoneNumberSvc{
		syncFunc: func(ctx context.Context) (int, error) {
			return 3, nil
		},
	}
	h := NewPhoneNumberHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/phone-numbers/sync", nil)
	w := httptest.NewRecorder()
	h.sync(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()

	data := body["data"].(map[string]interface{})
	if data["synced"] != 3.0 {
		t.Fatalf("synced = %v, want 3", data["synced"])
	}
}

func TestPhoneNumberHandler_Sync_Error(t *testing.T) {
	svc := &mockPhoneNumberSvc{
		syncFunc: func(ctx context.Context) (int, error) {
			return 0, errors.New("meta API error")
		},
	}
	h := NewPhoneNumberHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/phone-numbers/sync", nil)
	w := httptest.NewRecorder()
	h.sync(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func noopAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func TestPhoneNumberHandler_Routes(t *testing.T) {
	svc := &mockPhoneNumberSvc{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{
				{PhoneNumber: "+62811", PhoneNumberID: "pn1", IsActive: true},
			}, nil
		},
		syncFunc: func(ctx context.Context) (int, error) {
			return 2, nil
		},
		getProfFunc: func(ctx context.Context, id string) (*models.WhatsAppBusinessProfile, error) {
			return &models.WhatsAppBusinessProfile{}, nil
		},
	}

	h := NewPhoneNumberHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux, noopAuth)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("list", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/phone-numbers")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("sync", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/phone-numbers", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("assign-requires-auth", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/phone-numbers/pn1/assign", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("profile-get-requires-auth", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/phone-numbers/pn1/profile")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}
