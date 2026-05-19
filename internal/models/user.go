package models

import (
	"context"
	"time"
)

// User represents an authenticated user within a company.
type User struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserRole defines the role type for authorization.
type UserRole string

const (
	RoleAdmin      UserRole = "admin"
	RoleSuperadmin UserRole = "superadmin"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByCompanyID(ctx context.Context, companyID string) ([]User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}
