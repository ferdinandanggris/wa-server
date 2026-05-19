package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/wa-server/internal/models"
)

// BillingRepository implements persistence for billing logs.
type BillingRepository struct {
	db *DB
}

// NewBillingRepository creates a new BillingRepository.
func NewBillingRepository(db *DB) *BillingRepository {
	return &BillingRepository{db: db}
}

func (r *BillingRepository) Create(ctx context.Context, log *models.BillingLog) error {
	if log.ID == "" {
		log.ID = generateUUID()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO billing_logs (id, company_id, template_id, conversation_id, message_id, template_cost, phone_number, conversation_category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.CompanyID,
		log.TemplateID,
		log.ConversationID,
		log.MessageID,
		log.TemplateCost,
		log.PhoneNumber,
		log.ConversationCategory,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create billing log: %w", err)
	}

	return nil
}

func (r *BillingRepository) GetByCompanyID(ctx context.Context, companyID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	query := `
		SELECT id, company_id, COALESCE(template_id, ''), COALESCE(conversation_id, ''), COALESCE(message_id, ''),
			template_cost, COALESCE(phone_number, ''), COALESCE(conversation_category, ''), created_at
		FROM billing_logs
		WHERE company_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, companyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing logs: %w", err)
	}
	defer rows.Close()

	var logs []models.BillingLog
	for rows.Next() {
		var l models.BillingLog
		if err := rows.Scan(&l.ID, &l.CompanyID, &l.TemplateID, &l.ConversationID, &l.MessageID,
			&l.TemplateCost, &l.PhoneNumber, &l.ConversationCategory, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan billing log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, rows.Err()
}

func (r *BillingRepository) GetByCompanyIDAndPhone(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	query := `
		SELECT id, company_id, COALESCE(template_id, ''), COALESCE(conversation_id, ''), COALESCE(message_id, ''),
			template_cost, COALESCE(phone_number, ''), COALESCE(conversation_category, ''), created_at
		FROM billing_logs
		WHERE company_id = $1 AND phone_number = $2 AND created_at >= $3 AND created_at <= $4
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, companyID, phoneNumber, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing logs by phone: %w", err)
	}
	defer rows.Close()

	var logs []models.BillingLog
	for rows.Next() {
		var l models.BillingLog
		if err := rows.Scan(&l.ID, &l.CompanyID, &l.TemplateID, &l.ConversationID, &l.MessageID,
			&l.TemplateCost, &l.PhoneNumber, &l.ConversationCategory, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan billing log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, rows.Err()
}

func (r *BillingRepository) GetTemplateUsage(ctx context.Context, companyID, templateID string, startDate, endDate time.Time) ([]models.BillingLog, error) {
	query := `
		SELECT id, company_id, COALESCE(template_id, ''), COALESCE(conversation_id, ''), COALESCE(message_id, ''),
			template_cost, COALESCE(phone_number, ''), COALESCE(conversation_category, ''), created_at
		FROM billing_logs
		WHERE company_id = $1 AND template_id = $2 AND created_at >= $3 AND created_at <= $4
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, companyID, templateID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get template usage: %w", err)
	}
	defer rows.Close()

	var logs []models.BillingLog
	for rows.Next() {
		var l models.BillingLog
		if err := rows.Scan(&l.ID, &l.CompanyID, &l.TemplateID, &l.ConversationID, &l.MessageID,
			&l.TemplateCost, &l.PhoneNumber, &l.ConversationCategory, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan billing log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, rows.Err()
}

func (r *BillingRepository) GetTotalUsage(ctx context.Context, companyID string) (int, error) {
	query := `SELECT COUNT(*) FROM billing_logs WHERE company_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, companyID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total usage: %w", err)
	}

	return count, nil
}

func (r *BillingRepository) GetUsageByDate(ctx context.Context, companyID string, date time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM billing_logs WHERE company_id = $1 AND created_at::date = $2::date`

	var count int
	err := r.db.QueryRowContext(ctx, query, companyID, date).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get usage by date: %w", err)
	}

	return count, nil
}

func (r *BillingRepository) GetCostByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.BillingCostSummary, error) {
	query := `
		SELECT COALESCE(phone_number, ''), COALESCE(conversation_category, ''), COUNT(*), SUM(template_cost)
		FROM billing_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY phone_number, conversation_category
		ORDER BY phone_number, conversation_category
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by date range: %w", err)
	}
	defer rows.Close()

	var summaries []models.BillingCostSummary
	for rows.Next() {
		var s models.BillingCostSummary
		if err := rows.Scan(&s.PhoneNumber, &s.ConversationCategory, &s.TotalMessages, &s.TotalCost); err != nil {
			return nil, fmt.Errorf("failed to scan billing cost summary: %w", err)
		}
		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

func (r *BillingRepository) UpdateCost(ctx context.Context, id string, cost float64) error {
	query := `UPDATE billing_logs SET template_cost = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, cost)
	if err != nil {
		return fmt.Errorf("failed to update billing cost: %w", err)
	}

	return nil
}
