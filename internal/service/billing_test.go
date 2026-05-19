package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wa-server/internal/models"
)

// ---- mocks ----

type mockBillingRepo struct {
	createFunc                 func(ctx context.Context, log *models.BillingLog) error
	getByCompanyIDFunc         func(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error)
	getByCompanyIDAndPhoneFunc func(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error)
	getCostByDateRangeFunc     func(ctx context.Context, startDate, endDate time.Time) ([]models.BillingCostSummary, error)
	updateCostFunc             func(ctx context.Context, id string, cost float64) error
}

func (m *mockBillingRepo) Create(ctx context.Context, log *models.BillingLog) error {
	return m.createFunc(ctx, log)
}
func (m *mockBillingRepo) GetByCompanyID(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	return m.getByCompanyIDFunc(ctx, companyID, startDate, endDate)
}
func (m *mockBillingRepo) GetByCompanyIDAndPhone(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	return m.getByCompanyIDAndPhoneFunc(ctx, companyID, phoneNumber, startDate, endDate)
}
func (m *mockBillingRepo) GetCostByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.BillingCostSummary, error) {
	return m.getCostByDateRangeFunc(ctx, startDate, endDate)
}
func (m *mockBillingRepo) UpdateCost(ctx context.Context, id string, cost float64) error {
	return m.updateCostFunc(ctx, id, cost)
}

type mockCompanyRepo struct {
	getByIDFunc           func(ctx context.Context, id string) (*models.Company, error)
	getByPhoneNumberFunc  func(ctx context.Context, phoneNumber string) (*models.Company, error)
	tryIncrementQuotaFunc func(ctx context.Context, id string, amount int) (bool, error)
	decrementQuotaFunc    func(ctx context.Context, id string, amount int) error
}

func (m *mockCompanyRepo) GetByID(ctx context.Context, id string) (*models.Company, error) {
	return m.getByIDFunc(ctx, id)
}
func (m *mockCompanyRepo) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Company, error) {
	return m.getByPhoneNumberFunc(ctx, phoneNumber)
}
func (m *mockCompanyRepo) TryIncrementQuota(ctx context.Context, id string, amount int) (bool, error) {
	return m.tryIncrementQuotaFunc(ctx, id, amount)
}
func (m *mockCompanyRepo) DecrementQuota(ctx context.Context, id string, amount int) error {
	return m.decrementQuotaFunc(ctx, id, amount)
}

type mockWhatsapp struct {
	getConversationAnalyticsFunc func(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error)
}

func (m *mockWhatsapp) GetConversationAnalytics(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error) {
	return m.getConversationAnalyticsFunc(ctx, start, end, granularity)
}

// ---- tests ----

func TestBillingService_GetUsage(t *testing.T) {
	now := time.Now()
	logs := []models.BillingLog{{ID: "1", CompanyID: "c1", CreatedAt: now}}

	b := &mockBillingRepo{
		getByCompanyIDFunc: func(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			return logs, nil
		},
	}
	svc := NewBillingService(b, &mockCompanyRepo{}, &mockWhatsapp{})

	got, err := svc.GetUsage(context.Background(), "c1", now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("got %+v, want 1 log with ID 1", got)
	}
}

func TestBillingService_GetUsage_Error(t *testing.T) {
	b := &mockBillingRepo{
		getByCompanyIDFunc: func(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewBillingService(b, &mockCompanyRepo{}, &mockWhatsapp{})

	_, err := svc.GetUsage(context.Background(), "c1", time.Now().Add(-24*time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBillingService_GetQuota(t *testing.T) {
	company := &models.Company{ID: "c1", QuotaLimit: 1000, QuotaUsed: 50}

	c := &mockCompanyRepo{
		getByIDFunc: func(ctx context.Context, id string) (*models.Company, error) {
			return company, nil
		},
	}
	svc := NewBillingService(&mockBillingRepo{}, c, &mockWhatsapp{})

	got, err := svc.GetQuota(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c1" || got.QuotaLimit != 1000 || got.QuotaUsed != 50 {
		t.Fatalf("got %+v", got)
	}
}

func TestBillingService_GetQuota_NotFound(t *testing.T) {
	c := &mockCompanyRepo{
		getByIDFunc: func(ctx context.Context, id string) (*models.Company, error) {
			return nil, errors.New("not found")
		},
	}
	svc := NewBillingService(&mockBillingRepo{}, c, &mockWhatsapp{})

	_, err := svc.GetQuota(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBillingService_CreateBillingLog(t *testing.T) {
	b := &mockBillingRepo{
		createFunc: func(ctx context.Context, log *models.BillingLog) error {
			if log.CompanyID != "c1" {
				t.Errorf("CompanyID = %q, want c1", log.CompanyID)
			}
			return nil
		},
	}
	svc := NewBillingService(b, &mockCompanyRepo{}, &mockWhatsapp{})

	err := svc.CreateBillingLog(context.Background(), &models.BillingLog{CompanyID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBillingService_GetCostSummary(t *testing.T) {
	summary := []models.BillingCostSummary{
		{PhoneNumber: "+62811", TotalMessages: 10, TotalCost: 0.50},
	}

	b := &mockBillingRepo{
		getCostByDateRangeFunc: func(ctx context.Context, startDate, endDate time.Time) ([]models.BillingCostSummary, error) {
			return summary, nil
		},
	}
	svc := NewBillingService(b, &mockCompanyRepo{}, &mockWhatsapp{})

	got, err := svc.GetCostSummary(context.Background(), time.Now().Add(-7*24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TotalCost != 0.50 {
		t.Fatalf("got %+v", got)
	}
}

func TestBillingService_SyncCostsFromMeta(t *testing.T) {
	now := time.Now()
	analytics := &models.ConversationAnalyticsResponse{
		Data: []models.ConversationAnalyticsDataPoint{
			{PhoneNumber: "+62811", ConversationCategory: "marketing", Cost: 0.75, Conversation: 1},
			{PhoneNumber: "+62822", ConversationCategory: "service", Cost: 0.50, Conversation: 2},
		},
	}

	billingLogsByPhone := map[string][]models.BillingLog{
		"+62811": {{ID: "bl1", CompanyID: "c1", PhoneNumber: "+62811"}},
		"+62822": {{ID: "bl2", CompanyID: "c1", PhoneNumber: "+62822"}},
	}

	updatedIDs := make(map[string]bool)

	b := &mockBillingRepo{
		getByCompanyIDAndPhoneFunc: func(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			return billingLogsByPhone[phoneNumber], nil
		},
		updateCostFunc: func(ctx context.Context, id string, cost float64) error {
			updatedIDs[id] = true
			return nil
		},
	}

	c := &mockCompanyRepo{
		getByPhoneNumberFunc: func(ctx context.Context, phoneNumber string) (*models.Company, error) {
			return &models.Company{ID: "c1"}, nil
		},
	}

	w := &mockWhatsapp{
		getConversationAnalyticsFunc: func(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error) {
			if granularity != "DAY" {
				t.Errorf("granularity = %q, want DAY", granularity)
			}
			return analytics, nil
		},
	}

	svc := NewBillingService(b, c, w)
	updated, err := svc.SyncCostsFromMeta(context.Background(), now.Add(-7*24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}

	if updated != 2 {
		t.Fatalf("updated = %d, want 2", updated)
	}
	if !updatedIDs["bl1"] || !updatedIDs["bl2"] {
		t.Fatal("expected both billing logs to be updated")
	}
}

func TestBillingService_SyncCostsFromMeta_APIError(t *testing.T) {
	w := &mockWhatsapp{
		getConversationAnalyticsFunc: func(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error) {
			return nil, errors.New("meta API error")
		},
	}
	svc := NewBillingService(&mockBillingRepo{}, &mockCompanyRepo{}, w)

	_, err := svc.SyncCostsFromMeta(context.Background(), time.Now().Add(-7*24*time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error from Meta API")
	}
}

func TestBillingService_SyncCostsFromMeta_SkipsEmptyPhone(t *testing.T) {
	analytics := &models.ConversationAnalyticsResponse{
		Data: []models.ConversationAnalyticsDataPoint{
			{PhoneNumber: "", ConversationCategory: "service", Cost: 0.50},
			{PhoneNumber: "+62811", ConversationCategory: "marketing", Cost: 0.75, Conversation: 1},
		},
	}

	b := &mockBillingRepo{
		getByCompanyIDAndPhoneFunc: func(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			if phoneNumber != "+62811" {
				t.Fatalf("unexpected phone: %q", phoneNumber)
			}
			return []models.BillingLog{{ID: "bl1", PhoneNumber: "+62811"}}, nil
		},
		updateCostFunc: func(ctx context.Context, id string, cost float64) error {
			return nil
		},
	}

	c := &mockCompanyRepo{
		getByPhoneNumberFunc: func(ctx context.Context, phoneNumber string) (*models.Company, error) {
			return &models.Company{ID: "c1"}, nil
		},
	}

	w := &mockWhatsapp{
		getConversationAnalyticsFunc: func(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error) {
			return analytics, nil
		},
	}

	svc := NewBillingService(b, c, w)
	updated, err := svc.SyncCostsFromMeta(context.Background(), time.Now().Add(-7*24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if updated != 1 {
		t.Fatalf("updated = %d, want 1 (empty phone should be skipped)", updated)
	}
}

func TestBillingService_SyncCostsFromMeta_NoMatchingCompany(t *testing.T) {
	analytics := &models.ConversationAnalyticsResponse{
		Data: []models.ConversationAnalyticsDataPoint{
			{PhoneNumber: "+62811", ConversationCategory: "service", Cost: 0.50, Conversation: 1},
		},
	}

	b := &mockBillingRepo{
		getByCompanyIDAndPhoneFunc: func(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error) {
			return nil, nil
		},
	}

	c := &mockCompanyRepo{
		getByPhoneNumberFunc: func(ctx context.Context, phoneNumber string) (*models.Company, error) {
			return nil, errors.New("no company for this phone")
		},
	}

	w := &mockWhatsapp{
		getConversationAnalyticsFunc: func(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error) {
			return analytics, nil
		},
	}

	svc := NewBillingService(b, c, w)
	updated, err := svc.SyncCostsFromMeta(context.Background(), time.Now().Add(-7*24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if updated != 0 {
		t.Fatalf("updated = %d, want 0 (no matching company)", updated)
	}
}
