package models

import (
	"context"
	"time"
)

// BillingLog records a billing entry for message usage.
type BillingLog struct {
	ID                   string    `json:"id"`
	CompanyID            string    `json:"company_id"`
	TemplateID           string    `json:"template_id"`
	ConversationID       string    `json:"conversation_id,omitempty"`
	MessageID            string    `json:"message_id,omitempty"`
	TemplateCost         float64   `json:"template_cost"`
	PhoneNumber          string    `json:"phone_number,omitempty"`
	ConversationCategory string    `json:"conversation_category,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// BillingRepository defines persistence operations for billing logs.
type BillingRepository interface {
	Create(ctx context.Context, log *BillingLog) error
	GetByCompanyID(ctx context.Context, companyID string, startDate, endDate time.Time) ([]BillingLog, error)
	GetByCompanyIDAndPhone(ctx context.Context, companyID, phoneNumber string, startDate, endDate time.Time) ([]BillingLog, error)
	GetTemplateUsage(ctx context.Context, companyID, templateID string, startDate, endDate time.Time) ([]BillingLog, error)
	GetTotalUsage(ctx context.Context, companyID string) (int, error)
	GetUsageByDate(ctx context.Context, companyID string, date time.Time) (int, error)
	GetCostByDateRange(ctx context.Context, startDate, endDate time.Time) ([]BillingCostSummary, error)
	UpdateCost(ctx context.Context, id string, cost float64) error
}

// BillingCostSummary aggregates billing data by phone number and category.
type BillingCostSummary struct {
	PhoneNumber          string  `json:"phone_number"`
	ConversationCategory string  `json:"conversation_category"`
	TotalMessages        int     `json:"total_messages"`
	TotalCost            float64 `json:"total_cost"`
}

// ConversationAnalyticsResponse wraps the Meta API response.
type ConversationAnalyticsResponse struct {
	Data []ConversationAnalyticsDataPoint `json:"data"`
}

// ConversationAnalyticsDataPoint is a single cost data point from Meta.
type ConversationAnalyticsDataPoint struct {
	PhoneNumber          string  `json:"phone_number"`
	ConversationCategory string  `json:"conversation_category"`
	Cost                 float64 `json:"cost"`
	Conversation         int     `json:"conversation"`
}
