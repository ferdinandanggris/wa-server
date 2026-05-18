package models

import (
	"context"
	"time"
)

type BillingLog struct {
	ID             string    `json:"id"`
	CompanyID      string    `json:"company_id"`
	TemplateID     string    `json:"template_id"`
	ConversationID string    `json:"conversation_id,omitempty"`
	MessageID      string    `json:"message_id,omitempty"`
	TemplateCost   float64   `json:"template_cost"`
	CreatedAt      time.Time `json:"created_at"`
}

type BillingRepository interface {
	Create(ctx context.Context, log *BillingLog) error
	GetByCompanyID(ctx context.Context, companyID string, startDate, endDate time.Time) ([]BillingLog, error)
	GetTemplateUsage(ctx context.Context, companyID, templateID string, startDate, endDate time.Time) ([]BillingLog, error)
	GetTotalUsage(ctx context.Context, companyID string) (int, error)
	GetUsageByDate(ctx context.Context, companyID string, date time.Time) (int, error)
}
