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
		INSERT INTO phone_numbers (id, company_id, phone_number, phone_number_id, verified_name, is_active, created_at, updated_at)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (phone_number) DO UPDATE SET
			phone_number_id = EXCLUDED.phone_number_id,
			verified_name = EXCLUDED.verified_name,
			company_id = COALESCE(EXCLUDED.company_id, phone_numbers.company_id),
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		pn.ID, companyID, pn.PhoneNumber, pn.PhoneNumberID,
		pn.VerifiedName, pn.IsActive, pn.CreatedAt, pn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert phone number: %w", err)
	}
	return nil
}

func (r *PhoneNumberRepository) UpdateProfile(ctx context.Context, pn *models.PhoneNumber) error {
	now := time.Now().UTC()
	query := `
		UPDATE phone_numbers SET
			verified_name = $2, about = $3, address = $4, description = $5,
			email = $6, websites = $7, vertical = $8, profile_picture_url = $9,
			profile_synced_at = $10, updated_at = $11
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		pn.ID, pn.VerifiedName, pn.About, pn.Address, pn.Description,
		pn.Email, pn.Websites, pn.Vertical, pn.ProfilePictureURL,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to update phone number profile: %w", err)
	}
	return nil
}

func scanPhoneNumber(row interface{ Scan(...interface{}) error }) (*models.PhoneNumber, error) {
	var pn models.PhoneNumber
	var companyID, verifiedName, about, address, description, email, websites, vertical, profilePicURL sql.NullString
	var lastSync sql.NullInt64
	var profileSyncedAt sql.NullTime
	err := row.Scan(
		&pn.ID, &companyID, &pn.PhoneNumber, &pn.PhoneNumberID,
		&verifiedName, &pn.IsActive, &pn.CreatedAt, &pn.UpdatedAt, &lastSync,
		&about, &address, &description, &email, &websites, &vertical, &profilePicURL, &profileSyncedAt,
	)
	if err != nil {
		return nil, err
	}
	pn.CompanyID = companyID.String
	pn.VerifiedName = verifiedName.String
	pn.About = about.String
	pn.Address = address.String
	pn.Description = description.String
	pn.Email = email.String
	pn.Websites = websites.String
	pn.Vertical = vertical.String
	pn.ProfilePictureURL = profilePicURL.String
	if profileSyncedAt.Valid {
		pn.ProfileSyncedAt = &profileSyncedAt.Time
	}
	if lastSync.Valid {
		pn.LastSyncPricing = &lastSync.Int64
	}
	return &pn, nil
}

func (r *PhoneNumberRepository) GetByID(ctx context.Context, id string) (*models.PhoneNumber, error) {
	query := `SELECT ` + phoneNumberCols + ` FROM phone_numbers WHERE id = $1`
	return scanPhoneNumber(r.db.QueryRowContext(ctx, query, id))
}

func (r *PhoneNumberRepository) AssignCompany(ctx context.Context, id, companyID string) error {
	query := `UPDATE phone_numbers SET company_id = $1::uuid, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, companyID, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to assign company to phone number: %w", err)
	}
	return nil
}

func (r *PhoneNumberRepository) UpdateIsActive(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE phone_numbers SET is_active = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, isActive, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update is_active on phone number: %w", err)
	}
	return nil
}

const phoneNumberCols = `id, company_id, phone_number, phone_number_id, verified_name, is_active, created_at, updated_at, last_sync_pricing, about, address, description, email, websites, vertical, profile_picture_url, profile_synced_at`

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
