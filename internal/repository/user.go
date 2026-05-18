package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/wa-server/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, company_id, email, password_hash, name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.CompanyID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.CompanyID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.CompanyID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (r *UserRepo) GetByCompanyID(ctx context.Context, companyID string) ([]models.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE company_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.CompanyID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.Role,
			&user.IsActive,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET company_id = $2, email = $3, password_hash = $4, name = $5, role = $6, is_active = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.CompanyID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.IsActive,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
