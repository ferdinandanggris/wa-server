package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wa-server/internal/models"
)

type mockPricingRepo struct {
	upsertFunc func(ctx context.Context, p *models.WabaPricing) error
	getByPhone func(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error)
	getSummary func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error)
}

func (m *mockPricingRepo) UpsertPricing(ctx context.Context, p *models.WabaPricing) error {
	return m.upsertFunc(ctx, p)
}
func (m *mockPricingRepo) GetByPhoneNumber(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
	return m.getByPhone(ctx, phone, start, end)
}
func (m *mockPricingRepo) GetSummary(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
	return m.getSummary(ctx, start, end)
}

type mockPhonePricingRepo struct {
	listFunc       func(ctx context.Context) ([]models.PhoneNumber, error)
	updateSyncFunc func(ctx context.Context, unixSeconds int64) error
}

func (m *mockPhonePricingRepo) List(ctx context.Context) ([]models.PhoneNumber, error) {
	return m.listFunc(ctx)
}
func (m *mockPhonePricingRepo) UpdateLastSyncPricing(ctx context.Context, unixSeconds int64) error {
	return m.updateSyncFunc(ctx, unixSeconds)
}

type mockWhatsappPricing struct {
	getPricingFunc func(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error)
}

func (m *mockWhatsappPricing) GetPricingAnalytics(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error) {
	return m.getPricingFunc(ctx, start, end)
}

func TestPricingService_SyncFromMeta_WithExistingSync(t *testing.T) {
	lastSync := time.Now().Add(-24 * time.Hour).Unix()
	phoneRepo := &mockPhonePricingRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{
				{PhoneNumber: "+62811", LastSyncPricing: &lastSync},
			}, nil
		},
		updateSyncFunc: func(ctx context.Context, unixSeconds int64) error {
			return nil
		},
	}

	analytics := &models.PricingAnalyticsResponse{
		Data: []models.PricingAnalyticsEntry{
			{
				Name: "pricing_analytics",
				DataPoints: []models.PricingDataPoint{
					{Start: lastSync + 1, End: lastSync + 3600, PhoneNumber: "+62811", PricingCategory: "marketing", Volume: 10, Cost: 0.85},
				},
			},
		},
	}

	w := &mockWhatsappPricing{
		getPricingFunc: func(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error) {
			return analytics, nil
		},
	}

	upserted := 0
	repo := &mockPricingRepo{
		upsertFunc: func(ctx context.Context, p *models.WabaPricing) error {
			upserted++
			if p.WabaID != "waba_1" {
				t.Errorf("WabaID = %q, want waba_1", p.WabaID)
			}
			if p.PricingCategory != "marketing" {
				t.Errorf("PricingCategory = %q, want marketing", p.PricingCategory)
			}
			return nil
		},
	}

	svc := NewPricingService(repo, phoneRepo, w, "waba_1")
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
	if upserted != 1 {
		t.Fatalf("upserted = %d, want 1", upserted)
	}
}

func TestPricingService_SyncFromMeta_NoPreviousSync(t *testing.T) {
	phoneRepo := &mockPhonePricingRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{{PhoneNumber: "+62811", LastSyncPricing: nil}}, nil
		},
		updateSyncFunc: func(ctx context.Context, unixSeconds int64) error {
			return nil
		},
	}

	w := &mockWhatsappPricing{
		getPricingFunc: func(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error) {
			return &models.PricingAnalyticsResponse{
				Data: []models.PricingAnalyticsEntry{
					{
						Name: "pricing_analytics",
						DataPoints: []models.PricingDataPoint{
							{Start: end - 3600, End: end, PhoneNumber: "+62811", PricingCategory: "utility", Volume: 5, Cost: 0.35},
						},
					},
				},
			}, nil
		},
	}

	upserted := 0
	repo := &mockPricingRepo{
		upsertFunc: func(ctx context.Context, p *models.WabaPricing) error {
			upserted++
			return nil
		},
	}

	svc := NewPricingService(repo, phoneRepo, w, "waba_1")
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
}

func TestPricingService_SyncFromMeta_APIError(t *testing.T) {
	w := &mockWhatsappPricing{
		getPricingFunc: func(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error) {
			return nil, errors.New("meta API error")
		},
	}
	svc := NewPricingService(&mockPricingRepo{}, &mockPhonePricingRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{{PhoneNumber: "+62811"}}, nil
		},
	}, w, "waba_1")
	_, err := svc.SyncFromMeta(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPricingService_SyncFromMeta_SameDay(t *testing.T) {
	now := time.Now().Unix()
	phoneRepo := &mockPhonePricingRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{{PhoneNumber: "+62811", LastSyncPricing: &now}}, nil
		},
	}
	svc := NewPricingService(&mockPricingRepo{}, phoneRepo, &mockWhatsappPricing{}, "waba_1")
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if synced != 0 {
		t.Fatalf("synced = %d, want 0", synced)
	}
}

func TestPricingService_SyncFromMeta_PartialUpsertError(t *testing.T) {
	lastSync := time.Now().Add(-48 * time.Hour).Unix()
	phoneRepo := &mockPhonePricingRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return []models.PhoneNumber{{PhoneNumber: "+62811", LastSyncPricing: &lastSync}}, nil
		},
		updateSyncFunc: func(ctx context.Context, unixSeconds int64) error {
			return nil
		},
	}

	attempts := 0
	repo := &mockPricingRepo{
		upsertFunc: func(ctx context.Context, p *models.WabaPricing) error {
			attempts++
			if p.PricingCategory == "marketing" {
				return errors.New("db error")
			}
			return nil
		},
	}

	w := &mockWhatsappPricing{
		getPricingFunc: func(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error) {
			return &models.PricingAnalyticsResponse{
				Data: []models.PricingAnalyticsEntry{
					{
						Name: "pricing_analytics",
						DataPoints: []models.PricingDataPoint{
							{PhoneNumber: "+62811", PricingCategory: "marketing", Volume: 10, Cost: 0.85},
							{PhoneNumber: "+62811", PricingCategory: "utility", Volume: 5, Cost: 0.35},
						},
					},
				},
			}, nil
		},
	}

	svc := NewPricingService(repo, phoneRepo, w, "waba_1")
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal("should not error on partial upsert failure")
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1 (second upsert succeeded)", synced)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestPricingService_GetUsage(t *testing.T) {
	now := time.Now()
	expected := []models.WabaPricing{
		{PhoneNumber: "+62811", PricingCategory: "marketing", Volume: 10, Cost: 0.85},
	}
	repo := &mockPricingRepo{
		getByPhone: func(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
			return expected, nil
		},
	}
	svc := NewPricingService(repo, &mockPhonePricingRepo{}, &mockWhatsappPricing{}, "waba_1")
	got, err := svc.GetUsage(context.Background(), "+62811", now.Add(-7*24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
}

func TestPricingService_GetSummary(t *testing.T) {
	now := time.Now()
	expected := []models.PricingSummary{
		{Month: "2026-05", PricingCategory: "marketing", TotalVolume: 100, TotalCost: 8.50},
	}
	repo := &mockPricingRepo{
		getSummary: func(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
			return expected, nil
		},
	}
	svc := NewPricingService(repo, &mockPhonePricingRepo{}, &mockWhatsappPricing{}, "waba_1")
	got, err := svc.GetSummary(context.Background(), now.Add(-30*24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TotalCost != 8.50 {
		t.Fatalf("got %+v", got)
	}
}
