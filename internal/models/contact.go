package models

import (
	"context"
	"time"
)

// Contact represents a WhatsApp contact associated with a company.
type Contact struct {
	ID                string    `json:"id"`
	CompanyID         string    `json:"company_id"`
	WAID              string    `json:"wa_id"`
	PhoneNumber       string    `json:"phone_number"`
	Name              string    `json:"name,omitempty"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
	IsBlocked         bool      `json:"is_blocked"`
	LastSeenAt        time.Time `json:"last_seen_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ContactRepository defines persistence operations for contacts.
type ContactRepository interface {
	Create(ctx context.Context, contact *Contact) error
	GetByID(ctx context.Context, id string) (*Contact, error)
	GetByWAID(ctx context.Context, companyID, waID string) (*Contact, error)
	GetByPhoneNumber(ctx context.Context, companyID, phoneNumber string) (*Contact, error)
	Update(ctx context.Context, contact *Contact) error
	Upsert(ctx context.Context, contact *Contact) error
	Delete(ctx context.Context, id string) error
	ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]Contact, error)
}
