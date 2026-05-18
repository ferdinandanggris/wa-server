package models

import (
	"context"
	"time"
)

type Conversation struct {
	ID                    string     `json:"id"`
	CompanyID             string     `json:"company_id"`
	ContactID             string     `json:"contact_id"`
	AssignedAgentID       string     `json:"assigned_agent_id,omitempty"`
	Status                string     `json:"status"`
	LastCustomerMessageAt *time.Time `json:"last_customer_message_at,omitempty"`
	LastAgentMessageAt    *time.Time `json:"last_agent_message_at,omitempty"`
	Is24hWindowActive     bool       `json:"is_24h_window_active"`
	UnreadCount           int        `json:"unread_count"`
	LastMessagePreview    string     `json:"last_message_preview,omitempty"`
	StartedAt             time.Time  `json:"started_at"`
	ClosedAt              *time.Time `json:"closed_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type ConversationStatus string

const (
	ConversationStatusOpen      ConversationStatus = "open"
	ConversationStatusAssigned  ConversationStatus = "assigned"
	ConversationStatusClosed    ConversationStatus = "closed"
	ConversationStatusEscalated ConversationStatus = "escalated"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *Conversation) error
	GetByID(ctx context.Context, id string) (*Conversation, error)
	GetByIDWithCompany(ctx context.Context, id string) (*Conversation, error)
	GetByContactID(ctx context.Context, companyID, contactID string) (*Conversation, error)
	Update(ctx context.Context, conv *Conversation) error
	Update24hWindow(ctx context.Context, id string, isActive bool, lastMessageAt time.Time) error
	AssignAgent(ctx context.Context, id, agentID string) error
	IncrementUnread(ctx context.Context, id string) error
	ResetUnread(ctx context.Context, id string) error
	ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]Conversation, error)
	ListByAgent(ctx context.Context, agentID string) ([]Conversation, error)
	ListOpen(ctx context.Context, companyID string) ([]Conversation, error)
}
