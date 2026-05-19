package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/wa-server/internal/models"
)

type WabaPricingRepo interface {
	UpsertPricing(ctx context.Context, p *models.WabaPricing) error
	GetByPhoneNumber(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error)
	GetSummary(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error)
}

type PhoneNumberRepoForPricing interface {
	List(ctx context.Context) ([]models.PhoneNumber, error)
	UpdateLastSyncPricing(ctx context.Context, unixSeconds int64) error
}

type WhatsAppClientForPricing interface {
	GetPricingAnalytics(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error)
}

type PricingService struct {
	pricingRepo WabaPricingRepo
	phoneRepo   PhoneNumberRepoForPricing
	whatsapp    WhatsAppClientForPricing
	wabaID      string
}

func NewPricingService(pricingRepo WabaPricingRepo, phoneRepo PhoneNumberRepoForPricing, whatsapp WhatsAppClientForPricing, wabaID string) *PricingService {
	return &PricingService{pricingRepo: pricingRepo, phoneRepo: phoneRepo, whatsapp: whatsapp, wabaID: wabaID}
}

func (s *PricingService) SyncFromMeta(ctx context.Context) (int, error) {
	numbers, err := s.phoneRepo.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list phone numbers: %w", err)
	}

	now := time.Now().UTC()
	end := now.Unix()

	var start int64
	for _, pn := range numbers {
		if pn.LastSyncPricing != nil && *pn.LastSyncPricing > start {
			start = *pn.LastSyncPricing
		}
	}
	if start == 0 {
		start = now.AddDate(0, 0, -30).Unix()
	}

	if time.Unix(start, 0).Truncate(24*time.Hour) == time.Unix(end, 0).Truncate(24*time.Hour) {
		slog.Info("pricing sync skipped - already up to date")
		return 0, nil
	}

	analytics, err := s.whatsapp.GetPricingAnalytics(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to get pricing analytics: %w", err)
	}

	synced := 0
	for _, entry := range analytics.Data {
		for _, dp := range entry.DataPoints {
			p := &models.WabaPricing{
				WabaID:          s.wabaID,
				PhoneNumber:     dp.PhoneNumber,
				PricingCategory: dp.PricingCategory,
				StartTime:       time.Unix(dp.Start, 0).UTC(),
				EndTime:         time.Unix(dp.End, 0).UTC(),
				Volume:          dp.Volume,
				Cost:            math.Round(dp.Cost*10000) / 10000,
			}
			if err := s.pricingRepo.UpsertPricing(ctx, p); err != nil {
				slog.Error("failed to upsert pricing data point", "phone", dp.PhoneNumber, "category", dp.PricingCategory, "error", err)
				continue
			}
			synced++
		}
	}

	var maxEnd int64
	if len(analytics.Data) > 0 && len(analytics.Data[0].DataPoints) > 0 {
		for _, entry := range analytics.Data {
			for _, dp := range entry.DataPoints {
				if dp.End > maxEnd {
					maxEnd = dp.End
				}
			}
		}
	}
	if maxEnd > 0 {
		if err := s.phoneRepo.UpdateLastSyncPricing(ctx, maxEnd); err != nil {
			slog.Error("failed to update last_sync_pricing", "error", err)
		}
	}

	slog.Info("pricing sync completed", "synced", synced)
	return synced, nil
}

func (s *PricingService) GetUsage(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
	return s.pricingRepo.GetByPhoneNumber(ctx, phone, start, end)
}

func (s *PricingService) GetSummary(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
	return s.pricingRepo.GetSummary(ctx, start, end)
}
