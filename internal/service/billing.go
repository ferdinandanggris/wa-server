package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wa-server/internal/models"
)

// BillingRepository defines persistence operations needed by billing service.
type BillingRepository interface {
	Create(ctx context.Context, log *models.BillingLog) error
	GetByCompanyID(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error)
	GetByCompanyIDAndPhone(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error)
	GetCostByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.BillingCostSummary, error)
	UpdateCost(ctx context.Context, id string, cost float64) error
}

// CompanyRepositoryForBilling isolates company methods needed by billing service.
type CompanyRepositoryForBilling interface {
	GetByID(ctx context.Context, id string) (*models.Company, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Company, error)
	TryIncrementQuota(ctx context.Context, id string, amount int) (bool, error)
	DecrementQuota(ctx context.Context, id string, amount int) error
}

// WhatsAppClientForBilling isolates WhatsApp client methods needed by billing service.
type WhatsAppClientForBilling interface {
	GetConversationAnalytics(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error)
}

// BillingService handles billing operations: usage, quota, cost sync.
type BillingService struct {
	billingRepo BillingRepository
	companyRepo CompanyRepositoryForBilling
	whatsapp    WhatsAppClientForBilling
}

// NewBillingService creates a new BillingService.
func NewBillingService(billingRepo BillingRepository, companyRepo CompanyRepositoryForBilling, whatsapp WhatsAppClientForBilling) *BillingService {
	return &BillingService{
		billingRepo: billingRepo,
		companyRepo: companyRepo,
		whatsapp:    whatsapp,
	}
}

// GetUsage returns billing logs for a company within a date range.
func (s *BillingService) GetUsage(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	return s.billingRepo.GetByCompanyID(ctx, companyID, startDate, endDate)
}

// GetQuota returns the company's current quota usage and limit.
func (s *BillingService) GetQuota(ctx context.Context, companyID string) (*models.Company, error) {
	return s.companyRepo.GetByID(ctx, companyID)
}

// SyncCostsFromMeta pulls actual costs from Meta analytics and updates billing logs.
func (s *BillingService) SyncCostsFromMeta(ctx context.Context, start, end time.Time) (int, error) {
	analytics, err := s.whatsapp.GetConversationAnalytics(ctx, start, end, "DAY")
	if err != nil {
		return 0, fmt.Errorf("failed to get analytics from Meta: %w", err)
	}

	updated := 0
	for _, dp := range analytics.Data {
		phone := dp.PhoneNumber
		if phone == "" {
			continue
		}

		company, err := s.companyRepo.GetByPhoneNumber(ctx, phone)
		if err != nil {
			slog.Warn("no company found for phone number in analytics", "phone", phone)
			continue
		}

		logs, err := s.billingRepo.GetByCompanyIDAndPhone(ctx, company.ID, phone, start, end)
		if err != nil {
			slog.Error("failed to get billing logs for sync", "company_id", company.ID, "error", err)
			continue
		}

		for _, log := range logs {
			if err := s.billingRepo.UpdateCost(ctx, log.ID, dp.Cost); err != nil {
				slog.Error("failed to update billing cost", "log_id", log.ID, "error", err)
				continue
			}
			updated++
		}
	}

	slog.Info("billing cost sync completed", "analytics_points", len(analytics.Data), "updated", updated)
	return updated, nil
}

// GetCostSummary aggregates billing costs by phone and category.
func (s *BillingService) GetCostSummary(ctx context.Context, start, end time.Time) ([]models.BillingCostSummary, error) {
	return s.billingRepo.GetCostByDateRange(ctx, start, end)
}

// CreateBillingLog persists a new billing log entry.
func (s *BillingService) CreateBillingLog(ctx context.Context, log *models.BillingLog) error {
	return s.billingRepo.Create(ctx, log)
}
