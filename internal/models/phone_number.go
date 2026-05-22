package models

import (
	"context"
	"time"
)

type PhoneNumber struct {
	ID                string    `json:"id"`
	CompanyID         string    `json:"company_id,omitempty"`
	PhoneNumber       string    `json:"phone_number"`
	PhoneNumberID     string    `json:"phone_number_id"`
	VerifiedName      string    `json:"verified_name,omitempty"`
	About             string    `json:"about,omitempty"`
	Address           string    `json:"address,omitempty"`
	Description       string    `json:"description,omitempty"`
	Email             string    `json:"email,omitempty"`
	Websites          string    `json:"websites,omitempty"`
	Vertical          string    `json:"vertical,omitempty"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
	ProfileSyncedAt   *time.Time `json:"profile_synced_at,omitempty"`
	IsActive          bool      `json:"is_active"`
	LastSyncPricing   *int64    `json:"last_sync_pricing,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PhoneNumberRepository interface {
	Upsert(ctx context.Context, pn *PhoneNumber) error
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*PhoneNumber, error)
	GetByMetaID(ctx context.Context, metaID string) (*PhoneNumber, error)
	GetByCompanyID(ctx context.Context, companyID string) ([]PhoneNumber, error)
	GetByID(ctx context.Context, id string) (*PhoneNumber, error)
	GetByConversationID(ctx context.Context, conversationID string) (*PhoneNumber, error)
	List(ctx context.Context) ([]PhoneNumber, error)
	UpdateProfile(ctx context.Context, pn *PhoneNumber) error
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
