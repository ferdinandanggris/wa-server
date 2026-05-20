// Package models provides domain models and repository interfaces for the WhatsApp gateway.
// This package defines entities for companies, users, agents, contacts, conversations, messages,
// templates, and billing - all with multi-tenant isolation support.
package models

import (
	"context"
	"database/sql"
	"time"
)

// Company represents a multi-tenant company in the WhatsApp gateway system.
// Each company has its own quota limit and is isolated by company_id in all queries.
type Company struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Code        string        `json:"code"`
	PhoneNumber sql.NullString `json:"phone_number,omitempty"`
	Address     sql.NullString `json:"address,omitempty"`
	IsActive    bool          `json:"is_active"`
	QuotaLimit  int           `json:"quota_limit"`
	QuotaUsed   int           `json:"quota_used"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CompanyRepository defines persistence operations for companies.
type CompanyRepository interface {
	Create(ctx context.Context, company *Company) error
	GetByID(ctx context.Context, id string) (*Company, error)
	GetByCode(ctx context.Context, code string) (*Company, error)
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Company, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*Company, error)
	IncrementQuota(ctx context.Context, id string, amount int) error
	TryIncrementQuota(ctx context.Context, id string, amount int) (bool, error)
	DecrementQuota(ctx context.Context, id string, amount int) error
}
