package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/wa-server/internal/models"
)

type PricingRepository struct {
	db *DB
}

func NewPricingRepository(db *DB) *PricingRepository {
	return &PricingRepository{db: db}
}

func (r *PricingRepository) UpsertPricing(ctx context.Context, p *models.WabaPricing) error {
	p.ID = generateUUID()
	p.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO waba_pricing (id, waba_id, phone_number, pricing_category, start_time, end_time, volume, cost, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (waba_id, phone_number, pricing_category, start_time) DO UPDATE SET
			volume = EXCLUDED.volume,
			cost = EXCLUDED.cost,
			end_time = EXCLUDED.end_time
	`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.WabaID, p.PhoneNumber, p.PricingCategory,
		p.StartTime, p.EndTime, p.Volume, p.Cost, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert pricing: %w", err)
	}
	return nil
}

func (r *PricingRepository) GetByPhoneNumber(ctx context.Context, phone string, start, end time.Time) ([]models.WabaPricing, error) {
	query := `
		SELECT id, waba_id, phone_number, pricing_category, start_time, end_time, volume, cost, created_at
		FROM waba_pricing
		WHERE phone_number = $1 AND start_time >= $2 AND end_time <= $3
		ORDER BY start_time DESC, pricing_category
	`

	rows, err := r.db.QueryContext(ctx, query, phone, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.WabaPricing
	for rows.Next() {
		var p models.WabaPricing
		if err := rows.Scan(&p.ID, &p.WabaID, &p.PhoneNumber, &p.PricingCategory,
			&p.StartTime, &p.EndTime, &p.Volume, &p.Cost, &p.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PricingRepository) GetSummary(ctx context.Context, start, end time.Time) ([]models.PricingSummary, error) {
	query := `
		SELECT
			to_char(start_time, 'YYYY-MM') AS month,
			pricing_category,
			SUM(volume) AS total_volume,
			SUM(cost) AS total_cost
		FROM waba_pricing
		WHERE start_time >= $1 AND end_time <= $2
		GROUP BY month, pricing_category
		ORDER BY month DESC, pricing_category
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PricingSummary
	for rows.Next() {
		var s models.PricingSummary
		if err := rows.Scan(&s.Month, &s.PricingCategory, &s.TotalVolume, &s.TotalCost); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
