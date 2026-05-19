package models

import (
	"context"
	"time"
)

// Template represents a WhatsApp message template.
type Template struct {
	ID             string    `json:"id"`
	WATemplateID   string    `json:"wa_template_id"`
	MetaName       string    `json:"meta_name,omitempty"`
	Name           string    `json:"name"`
	Language       string    `json:"language"`
	Category       string    `json:"category"`
	Content        string    `json:"content"`
	HeaderType     string    `json:"header_type,omitempty"`
	HeaderContent  string    `json:"header_content,omitempty"`
	BodyComponents string    `json:"body_components,omitempty"`
	FooterText     string    `json:"footer_text,omitempty"`
	Buttons        string    `json:"buttons,omitempty"`
	IsVerified     bool      `json:"is_verified"`
	MetaStatus     string    `json:"meta_status,omitempty"`
	CreatedBy      string    `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TemplateCategory represents the category of a WhatsApp template.
type TemplateCategory string

const (
	TemplateCategoryMarketing      TemplateCategory = "marketing"
	TemplateCategoryUtility        TemplateCategory = "utility"
	TemplateCategoryAuthentication TemplateCategory = "authentication"
)

// TemplateMetaStatus represents the approval status from Meta.
type TemplateMetaStatus string

const (
	TemplateMetaPending  TemplateMetaStatus = "PENDING"
	TemplateMetaApproved TemplateMetaStatus = "APPROVED"
	TemplateMetaRejected TemplateMetaStatus = "REJECTED"
)

// TemplateRepository defines persistence operations for templates.
type TemplateRepository interface {
	Create(ctx context.Context, tmpl *Template) error
	GetByID(ctx context.Context, id string) (*Template, error)
	GetByWATemplateID(ctx context.Context, waID string) (*Template, error)
	GetByMetaNameAndLanguage(ctx context.Context, metaName, language string) (*Template, error)
	GetByMetaStatus(ctx context.Context, status string) ([]Template, error)
	GetByName(ctx context.Context, name string) ([]Template, error)
	GetVerified(ctx context.Context) ([]Template, error)
	Update(ctx context.Context, tmpl *Template) error
	UpdateMetaStatus(ctx context.Context, id, metaStatus string, isVerified bool) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Template, error)
}
