package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wa-server/internal/models"
)

type ConversationRepository struct {
	db *DB
}

func NewConversationRepository(db *DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, conv *models.Conversation) error {
	if conv.ID == "" {
		conv.ID = generateUUID()
	}
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = time.Now().UTC()
	}
	if conv.UpdatedAt.IsZero() {
		conv.UpdatedAt = time.Now().UTC()
	}
	if conv.StartedAt.IsZero() {
		conv.StartedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO conversations (
			id, company_id, contact_id, assigned_agent_id, status,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, started_at, closed_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		conv.ID,
		conv.CompanyID,
		conv.ContactID,
		conv.AssignedAgentID,
		conv.Status,
		conv.LastCustomerMessageAt,
		conv.LastAgentMessageAt,
		conv.Is24hWindowActive,
		conv.UnreadCount,
		conv.LastMessagePreview,
		conv.StartedAt,
		conv.ClosedAt,
		conv.CreatedAt,
		conv.UpdatedAt,
	)

	return err
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	query := `
		SELECT id, company_id, contact_id, assigned_agent_id, status,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, started_at, closed_at, created_at, updated_at
		FROM conversations WHERE id = $1
	`

	var conv models.Conversation
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&conv.ID,
		&conv.CompanyID,
		&conv.ContactID,
		&conv.AssignedAgentID,
		&conv.Status,
		&conv.LastCustomerMessageAt,
		&conv.LastAgentMessageAt,
		&conv.Is24hWindowActive,
		&conv.UnreadCount,
		&conv.LastMessagePreview,
		&conv.StartedAt,
		&conv.ClosedAt,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) GetByIDWithCompany(ctx context.Context, id string) (*models.Conversation, error) {
	return r.GetByID(ctx, id)
}

func (r *ConversationRepository) GetByContactID(ctx context.Context, companyID, contactID string) (*models.Conversation, error) {
	query := `
		SELECT id, company_id, contact_id, assigned_agent_id, status,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, started_at, closed_at, created_at, updated_at
		FROM conversations
		WHERE company_id = $1 AND contact_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var conv models.Conversation
	err := r.db.QueryRowContext(ctx, query, companyID, contactID).Scan(
		&conv.ID,
		&conv.CompanyID,
		&conv.ContactID,
		&conv.AssignedAgentID,
		&conv.Status,
		&conv.LastCustomerMessageAt,
		&conv.LastAgentMessageAt,
		&conv.Is24hWindowActive,
		&conv.UnreadCount,
		&conv.LastMessagePreview,
		&conv.StartedAt,
		&conv.ClosedAt,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) Update(ctx context.Context, conv *models.Conversation) error {
	conv.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE conversations
		SET assigned_agent_id = $2, status = $3, last_customer_message_at = $4,
			last_agent_message_at = $5, is_24h_window_active = $6, unread_count = $7,
			last_message_preview = $8, closed_at = $9, updated_at = $10
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		conv.ID,
		conv.AssignedAgentID,
		conv.Status,
		conv.LastCustomerMessageAt,
		conv.LastAgentMessageAt,
		conv.Is24hWindowActive,
		conv.UnreadCount,
		conv.LastMessagePreview,
		conv.ClosedAt,
		conv.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ConversationRepository) Update24hWindow(ctx context.Context, id string, isActive bool, lastMessageAt time.Time) error {
	query := `
		UPDATE conversations
		SET is_24h_window_active = $2, last_customer_message_at = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, isActive, lastMessageAt, time.Now().UTC())
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ConversationRepository) AssignAgent(ctx context.Context, id, agentID string) error {
	query := `
		UPDATE conversations
		SET assigned_agent_id = $2, status = 'assigned', updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, agentID, time.Now().UTC())
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ConversationRepository) IncrementUnread(ctx context.Context, id string) error {
	query := `UPDATE conversations SET unread_count = unread_count + 1, updated_at = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, time.Now().UTC())
	return err
}

func (r *ConversationRepository) ResetUnread(ctx context.Context, id string) error {
	query := `UPDATE conversations SET unread_count = 0, updated_at = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, time.Now().UTC())
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ConversationRepository) ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]models.Conversation, error) {
	query := `
		SELECT id, company_id, contact_id, assigned_agent_id, status,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, started_at, closed_at, created_at, updated_at
		FROM conversations
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []models.Conversation
	for rows.Next() {
		var conv models.Conversation
		err := rows.Scan(
			&conv.ID,
			&conv.CompanyID,
			&conv.ContactID,
			&conv.AssignedAgentID,
			&conv.Status,
			&conv.LastCustomerMessageAt,
			&conv.LastAgentMessageAt,
			&conv.Is24hWindowActive,
			&conv.UnreadCount,
			&conv.LastMessagePreview,
			&conv.StartedAt,
			&conv.ClosedAt,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}

	return convs, rows.Err()
}

func (r *ConversationRepository) ListByAgent(ctx context.Context, agentID string) ([]models.Conversation, error) {
	query := `
		SELECT id, company_id, contact_id, assigned_agent_id, status,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, started_at, closed_at, created_at, updated_at
		FROM conversations
		WHERE assigned_agent_id = $1
		ORDER BY last_customer_message_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []models.Conversation
	for rows.Next() {
		var conv models.Conversation
		err := rows.Scan(
			&conv.ID,
			&conv.CompanyID,
			&conv.ContactID,
			&conv.AssignedAgentID,
			&conv.Status,
			&conv.LastCustomerMessageAt,
			&conv.LastAgentMessageAt,
			&conv.Is24hWindowActive,
			&conv.UnreadCount,
			&conv.LastMessagePreview,
			&conv.StartedAt,
			&conv.ClosedAt,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}

	return convs, rows.Err()
}

func (r *ConversationRepository) ListOpen(ctx context.Context, companyID string) ([]models.Conversation, error) {
	query := `
		SELECT id, company_id, contact_id, assigned_agent_id, status,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, started_at, closed_at, created_at, updated_at
		FROM conversations
		WHERE company_id = $1 AND status IN ('open', 'assigned', 'escalated')
		ORDER BY last_customer_message_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []models.Conversation
	for rows.Next() {
		var conv models.Conversation
		err := rows.Scan(
			&conv.ID,
			&conv.CompanyID,
			&conv.ContactID,
			&conv.AssignedAgentID,
			&conv.Status,
			&conv.LastCustomerMessageAt,
			&conv.LastAgentMessageAt,
			&conv.Is24hWindowActive,
			&conv.UnreadCount,
			&conv.LastMessagePreview,
			&conv.StartedAt,
			&conv.ClosedAt,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}

	return convs, rows.Err()
}
