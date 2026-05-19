package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wa-server/internal/models"
)

// ErrCompanyNotFound is returned when a company record is not found.
var ErrCompanyNotFound = errors.New("company not found")

// CompanyRepo implements company persistence for PostgreSQL.
type CompanyRepo struct {
	db *DB
}

// NewCompanyRepo creates a new CompanyRepo.
func NewCompanyRepo(db *DB) *CompanyRepo {
	return &CompanyRepo{db: db}
}

func (r *CompanyRepo) Create(ctx context.Context, company *models.Company) error {
	query := `
		INSERT INTO companies (id, name, code, phone_number, address, is_active, quota_limit, quota_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		company.ID,
		company.Name,
		company.Code,
		company.PhoneNumber,
		company.Address,
		company.IsActive,
		company.QuotaLimit,
		company.QuotaUsed,
		company.CreatedAt,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}

	return nil
}

func (r *CompanyRepo) GetByID(ctx context.Context, id string) (*models.Company, error) {
	query := `
		SELECT id, name, code, phone_number, address, is_active, quota_limit, quota_used, created_at, updated_at
		FROM companies
		WHERE id = $1
	`

	var company models.Company
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&company.ID,
		&company.Name,
		&company.Code,
		&company.PhoneNumber,
		&company.Address,
		&company.IsActive,
		&company.QuotaLimit,
		&company.QuotaUsed,
		&company.CreatedAt,
		&company.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, ErrCompanyNotFound
	}

	return &company, nil
}

func (r *CompanyRepo) GetByCode(ctx context.Context, code string) (*models.Company, error) {
	query := `
		SELECT id, name, code, phone_number, address, is_active, quota_limit, quota_used, created_at, updated_at
		FROM companies
		WHERE code = $1
	`

	var company models.Company
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&company.ID,
		&company.Name,
		&company.Code,
		&company.PhoneNumber,
		&company.Address,
		&company.IsActive,
		&company.QuotaLimit,
		&company.QuotaUsed,
		&company.CreatedAt,
		&company.UpdatedAt,
	)
	if err != nil {
		return nil, ErrCompanyNotFound
	}

	return &company, nil
}

func (r *CompanyRepo) Update(ctx context.Context, company *models.Company) error {
	query := `
		UPDATE companies
		SET name = $2, phone_number = $3, address = $4, is_active = $5, quota_limit = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		company.ID,
		company.Name,
		company.PhoneNumber,
		company.Address,
		company.IsActive,
		company.QuotaLimit,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

func (r *CompanyRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM companies WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

func (r *CompanyRepo) List(ctx context.Context, limit, offset int) ([]models.Company, error) {
	query := `
		SELECT id, name, code, phone_number, address, is_active, quota_limit, quota_used, created_at, updated_at
		FROM companies
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var company models.Company
		if err := rows.Scan(
			&company.ID,
			&company.Name,
			&company.Code,
			&company.PhoneNumber,
			&company.Address,
			&company.IsActive,
			&company.QuotaLimit,
			&company.QuotaUsed,
			&company.CreatedAt,
			&company.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan company: %w", err)
		}
		companies = append(companies, company)
	}

	return companies, nil
}

func (r *CompanyRepo) IncrementQuota(ctx context.Context, id string, amount int) error {
	query := `
		UPDATE companies
		SET quota_used = quota_used + $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, amount, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment quota: %w", err)
	}

	return nil
}

func (r *CompanyRepo) IncrementQuotaWithLock(ctx context.Context, id string, amount int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	query := `
		UPDATE companies
		SET quota_used = quota_used + $2, updated_at = $3
		WHERE id = $1
		FOR UPDATE
	`

	_, err = tx.ExecContext(ctx, query, id, amount, time.Now())
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to increment quota with lock: %w (rollback: %v)", err, rbErr)
		}
		return fmt.Errorf("failed to increment quota with lock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
