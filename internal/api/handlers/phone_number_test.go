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
	syncFunc func(ctx context.Context) (int, error)
	listFunc func(ctx context.Context) ([]models.PhoneNumber, error)
}

func (m *mockPhoneNumberSvc) SyncFromMeta(ctx context.Context) (int, error) {
	return m.syncFunc(ctx)
}
func (m *mockPhoneNumberSvc) List(ctx context.Context) ([]models.PhoneNumber, error) {
	return m.listFunc(ctx)
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

	var numbers []models.PhoneNumber
	json.NewDecoder(resp.Body).Decode(&numbers)
	resp.Body.Close()

	if len(numbers) != 1 || numbers[0].PhoneNumber != "+62811" {
		t.Fatalf("got %+v", numbers)
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

	if body["synced"] != 3.0 {
		t.Fatalf("synced = %v, want 3", body["synced"])
	}
}

func TestPhoneNumberHandler_Sync_MethodNotAllowed(t *testing.T) {
	h := NewPhoneNumberHandler(&mockPhoneNumberSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers/sync", nil)
	w := httptest.NewRecorder()
	h.sync(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	resp.Body.Close()
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
	}

	h := NewPhoneNumberHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

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

	t.Run("method-not-allowed", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/phone-numbers/other")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
	})
}
