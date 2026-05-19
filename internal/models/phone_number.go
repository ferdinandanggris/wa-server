package models

import (
	"context"
	"time"
)

type PhoneNumber struct {
	ID              string    `json:"id"`
	CompanyID       string    `json:"company_id,omitempty"`
	PhoneNumber     string    `json:"phone_number"`
	PhoneNumberID   string    `json:"phone_number_id"`
	IsActive        bool      `json:"is_active"`
	LastSyncPricing *int64    `json:"last_sync_pricing,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PhoneNumberRepository interface {
	Upsert(ctx context.Context, pn *PhoneNumber) error
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*PhoneNumber, error)
	GetByMetaID(ctx context.Context, metaID string) (*PhoneNumber, error)
	GetByCompanyID(ctx context.Context, companyID string) ([]PhoneNumber, error)
	List(ctx context.Context) ([]PhoneNumber, error)
}

type PhoneNumberRepoForSync interface {
	List(ctx context.Context) ([]PhoneNumber, error)
	UpdateLastSyncPricing(ctx context.Context, unixSeconds int64) error
}

type WhatsAppPhoneNumber struct {
	ID            string `json:"id"`
	DisplayNumber string `json:"display_phone_number"`
	VerifiedName  string `json:"verified_name"`
}
