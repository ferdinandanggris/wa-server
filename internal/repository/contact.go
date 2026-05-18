package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wa-server/internal/models"
)

type ContactRepository struct {
	db *DB
}

func NewContactRepository(db *DB) *ContactRepository {
	return &ContactRepository{db: db}
}

func (r *ContactRepository) Create(ctx context.Context, contact *models.Contact) error {
	if contact.ID == "" {
		contact.ID = generateUUID()
	}
	if contact.CreatedAt.IsZero() {
		contact.CreatedAt = time.Now().UTC()
	}
	if contact.UpdatedAt.IsZero() {
		contact.UpdatedAt = time.Now().UTC()
	}

	var companyID interface{}
	if contact.CompanyID != "" {
		companyID = contact.CompanyID
	}

	query := `
		INSERT INTO contacts (id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, last_seen_at, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		contact.ID,
		companyID,
		contact.WAID,
		contact.PhoneNumber,
		contact.Name,
		contact.ProfilePictureURL,
		contact.IsBlocked,
		contact.LastSeenAt,
		contact.CreatedAt,
		contact.UpdatedAt,
	)

	return err
}

func (r *ContactRepository) GetByID(ctx context.Context, id string) (*models.Contact, error) {
	query := `
		SELECT id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, last_seen_at, created_at, updated_at
		FROM contacts WHERE id = $1
	`

	var contact models.Contact
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&contact.ID,
		&contact.CompanyID,
		&contact.WAID,
		&contact.PhoneNumber,
		&contact.Name,
		&contact.ProfilePictureURL,
		&contact.IsBlocked,
		&contact.LastSeenAt,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) GetByWAID(ctx context.Context, companyID, waID string) (*models.Contact, error) {
	query := `
		SELECT id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, last_seen_at, created_at, updated_at
		FROM contacts WHERE company_id = $1 AND wa_id = $2
	`

	var contact models.Contact
	err := r.db.QueryRowContext(ctx, query, companyID, waID).Scan(
		&contact.ID,
		&contact.CompanyID,
		&contact.WAID,
		&contact.PhoneNumber,
		&contact.Name,
		&contact.ProfilePictureURL,
		&contact.IsBlocked,
		&contact.LastSeenAt,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) GetByPhoneNumber(ctx context.Context, companyID, phoneNumber string) (*models.Contact, error) {
	query := `
		SELECT id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, last_seen_at, created_at, updated_at
		FROM contacts WHERE company_id = $1 AND phone_number = $2
	`

	var contact models.Contact
	err := r.db.QueryRowContext(ctx, query, companyID, phoneNumber).Scan(
		&contact.ID,
		&contact.CompanyID,
		&contact.WAID,
		&contact.PhoneNumber,
		&contact.Name,
		&contact.ProfilePictureURL,
		&contact.IsBlocked,
		&contact.LastSeenAt,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) Update(ctx context.Context, contact *models.Contact) error {
	contact.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE contacts
		SET company_id = $2, wa_id = $3, phone_number = $4, name = $5, profile_picture_url = $6, is_blocked = $7, last_seen_at = $8, updated_at = $9
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		contact.ID,
		contact.CompanyID,
		contact.WAID,
		contact.PhoneNumber,
		contact.Name,
		contact.ProfilePictureURL,
		contact.IsBlocked,
		contact.LastSeenAt,
		contact.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ContactRepository) Upsert(ctx context.Context, contact *models.Contact) error {
	if contact.ID == "" {
		contact.ID = generateUUID()
	}
	if contact.CreatedAt.IsZero() {
		contact.CreatedAt = time.Now().UTC()
	}
	if contact.UpdatedAt.IsZero() {
		contact.UpdatedAt = time.Now().UTC()
	}

	var companyID interface{}
	if contact.CompanyID != "" {
		companyID = contact.CompanyID
	}

	query := `
		INSERT INTO contacts (id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, last_seen_at, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (company_id, wa_id) DO UPDATE SET
			phone_number = EXCLUDED.phone_number,
			name = EXCLUDED.name,
			profile_picture_url = EXCLUDED.profile_picture_url,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		contact.ID,
		companyID,
		contact.WAID,
		contact.PhoneNumber,
		contact.Name,
		contact.ProfilePictureURL,
		contact.IsBlocked,
		contact.LastSeenAt,
		contact.CreatedAt,
		contact.UpdatedAt,
	).Scan(&contact.ID, &contact.CreatedAt)

	return err
}

func (r *ContactRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM contacts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ContactRepository) ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]models.Contact, error) {
	query := `
		SELECT id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, last_seen_at, created_at, updated_at
		FROM contacts
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var contact models.Contact
		err := rows.Scan(
			&contact.ID,
			&contact.CompanyID,
			&contact.WAID,
			&contact.PhoneNumber,
			&contact.Name,
			&contact.ProfilePictureURL,
			&contact.IsBlocked,
			&contact.LastSeenAt,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}
