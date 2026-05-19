package models

import (
	"context"
	"time"
)

// Agent represents a customer service agent within a company.
type Agent struct {
	ID                 string    `json:"id"`
	CompanyID          string    `json:"company_id"`
	UserID             string    `json:"user_id,omitempty"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Status             string    `json:"status"`
	MaxConcurrentChats int       `json:"max_concurrent_chats"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AgentStatus represents the availability status of an agent.
type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
	AgentStatusBusy    AgentStatus = "busy"
	AgentStatusAway    AgentStatus = "away"
)

// AgentRepository defines persistence operations for agents.
type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id string) (*Agent, error)
	GetByCompanyID(ctx context.Context, companyID string) ([]Agent, error)
	GetAvailable(ctx context.Context, companyID string) ([]Agent, error)
	Update(ctx context.Context, agent *Agent) error
	UpdateStatus(ctx context.Context, id string, status string) error
	Delete(ctx context.Context, id string) error
}
