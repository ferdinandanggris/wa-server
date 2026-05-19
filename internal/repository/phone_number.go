package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wa-server/internal/models"
)

type PhoneNumberRepository struct {
	db *DB
}

func NewPhoneNumberRepository(db *DB) *PhoneNumberRepository {
	return &PhoneNumberRepository{db: db}
}

func (r *PhoneNumberRepository) Upsert(ctx context.Context, pn *models.PhoneNumber) error {
	if pn.ID == "" {
		pn.ID = generateUUID()
	}
	now := time.Now().UTC()
	if pn.CreatedAt.IsZero() {
		pn.CreatedAt = now
	}
	pn.UpdatedAt = now

	var companyID interface{}
	if pn.CompanyID != "" {
		companyID = pn.CompanyID
	}

	query := `
		INSERT INTO phone_numbers (id, company_id, phone_number, phone_number_id, is_active, created_at, updated_at)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7)
		ON CONFLICT (phone_number) DO UPDATE SET
			phone_number_id = EXCLUDED.phone_number_id,
			company_id = COALESCE(EXCLUDED.company_id, phone_numbers.company_id),
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		pn.ID, companyID, pn.PhoneNumber, pn.PhoneNumberID,
		pn.IsActive, pn.CreatedAt, pn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert phone number: %w", err)
	}
	return nil
}

func scanPhoneNumber(row interface{ Scan(...interface{}) error }) (*models.PhoneNumber, error) {
	var pn models.PhoneNumber
	var companyID sql.NullString
	var lastSync sql.NullInt64
	err := row.Scan(
		&pn.ID, &companyID, &pn.PhoneNumber, &pn.PhoneNumberID,
		&pn.IsActive, &pn.CreatedAt, &pn.UpdatedAt, &lastSync,
	)
	if err != nil {
		return nil, err
	}
	pn.CompanyID = companyID.String
	if lastSync.Valid {
		pn.LastSyncPricing = &lastSync.Int64
	}
	return &pn, nil
}

const phoneNumberCols = `id, company_id, phone_number, phone_number_id, is_active, created_at, updated_at, last_sync_pricing`

func (r *PhoneNumberRepository) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.PhoneNumber, error) {
	query := `SELECT ` + phoneNumberCols + ` FROM phone_numbers WHERE phone_number = $1`

	return scanPhoneNumber(r.db.QueryRowContext(ctx, query, phoneNumber))
}

func (r *PhoneNumberRepository) GetByMetaID(ctx context.Context, metaID string) (*models.PhoneNumber, error) {
	query := `SELECT ` + phoneNumberCols + ` FROM phone_numbers WHERE phone_number_id = $1`

	return scanPhoneNumber(r.db.QueryRowContext(ctx, query, metaID))
}

func (r *PhoneNumberRepository) GetByCompanyID(ctx context.Context, companyID string) ([]models.PhoneNumber, error) {
	query := `SELECT ` + phoneNumberCols + ` FROM phone_numbers WHERE company_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PhoneNumber
	for rows.Next() {
		pn, err := scanPhoneNumber(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *pn)
	}
	return result, rows.Err()
}

func (r *PhoneNumberRepository) UpdateLastSyncPricing(ctx context.Context, unixSeconds int64) error {
	query := `UPDATE phone_numbers SET last_sync_pricing = $1`
	_, err := r.db.ExecContext(ctx, query, unixSeconds)
	if err != nil {
		return fmt.Errorf("failed to update last_sync_pricing: %w", err)
	}
	return nil
}

func (r *PhoneNumberRepository) List(ctx context.Context) ([]models.PhoneNumber, error) {
	query := `SELECT ` + phoneNumberCols + ` FROM phone_numbers ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PhoneNumber
	for rows.Next() {
		pn, err := scanPhoneNumber(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *pn)
	}
	return result, rows.Err()
}
