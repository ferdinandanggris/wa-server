package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wa-server/internal/models"
)

type MessageRepository struct {
	db *DB
}

func nullJSON(s string) interface{} {
	if s == "" || s == "null" {
		return nil
	}
	return s
}

func NewMessageRepository(db *DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, msg *models.Message) error {
	// Let PostgreSQL generate the ID
	msg.ID = uuid.New().String()

	if msg.MessageID == "" {
		msg.MessageID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}

	slog.Info("message repo create", "id", msg.ID, "conversation_id", msg.ConversationID, "message_id", msg.MessageID)

	var templateID *string
	if msg.TemplateID != "" {
		templateID = &msg.TemplateID
	}

	query := `
		INSERT INTO messages (
			id, conversation_id, message_id, direction, message_type,
			content, template_id, template_params, media_url, status,
			wa_status, sent_at, delivered_at, read_at, error_message, created_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.db.ExecContext(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.MessageID,
		msg.Direction,
		msg.MessageType,
		msg.Content,
		templateID,
		nullJSON(msg.TemplateParams),
		msg.MediaURL,
		msg.Status,
		msg.WAStatus,
		msg.SentAt,
		msg.DeliveredAt,
		msg.ReadAt,
		msg.ErrorMessage,
		msg.CreatedAt,
	)

	if err != nil {
		slog.Error("DB insert error", "error", err, "templateID", templateID)
	}

	return err
}

func (r *MessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			content, template_id, template_params, media_url, status,
			wa_status, sent_at, delivered_at, read_at, error_message, created_at
		FROM messages WHERE id = $1
	`

	var msg models.Message
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.MessageID,
		&msg.Direction,
		&msg.MessageType,
		&msg.Content,
		&msg.TemplateID,
		&msg.TemplateParams,
		&msg.MediaURL,
		&msg.Status,
		&msg.WAStatus,
		&msg.SentAt,
		&msg.DeliveredAt,
		&msg.ReadAt,
		&msg.ErrorMessage,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *MessageRepository) GetByMessageID(ctx context.Context, messageID string) (*models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			content, template_id, template_params, media_url, status,
			wa_status, sent_at, delivered_at, read_at, error_message, created_at
		FROM messages WHERE message_id = $1
	`

	var msg models.Message
	err := r.db.QueryRowContext(ctx, query, messageID).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.MessageID,
		&msg.Direction,
		&msg.MessageType,
		&msg.Content,
		&msg.TemplateID,
		&msg.TemplateParams,
		&msg.MediaURL,
		&msg.Status,
		&msg.WAStatus,
		&msg.SentAt,
		&msg.DeliveredAt,
		&msg.ReadAt,
		&msg.ErrorMessage,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *MessageRepository) GetByConversationID(ctx context.Context, convID string, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			content, template_id, template_params, media_url, status,
			wa_status, sent_at, delivered_at, read_at, error_message, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, convID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.MessageID,
			&msg.Direction,
			&msg.MessageType,
			&msg.Content,
			&msg.TemplateID,
			&msg.TemplateParams,
			&msg.MediaURL,
			&msg.Status,
			&msg.WAStatus,
			&msg.SentAt,
			&msg.DeliveredAt,
			&msg.ReadAt,
			&msg.ErrorMessage,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (r *MessageRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE messages SET status = $1, error_message = '' WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *MessageRepository) UpdateWAMessageID(ctx context.Context, id, waMessageID string) error {
	query := `UPDATE messages SET message_id = $1, status = 'sent', sent_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, waMessageID, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type MessageFilter struct {
	CompanyID      string
	ConversationID string
	Status         string
	Direction      string
	From, To       time.Time
}

func (r *MessageRepository) List(ctx context.Context, filter MessageFilter, limit, offset int) ([]models.Message, error) {
	args := []interface{}{}
	conditions := []string{}

	if filter.CompanyID != "" {
		conditions = append(conditions, `c.company_id = $`+fmt.Sprint(len(args)+1))
		args = append(args, filter.CompanyID)
	}

	if filter.ConversationID != "" {
		conditions = append(conditions, `m.conversation_id = $`+fmt.Sprint(len(args)+1))
		args = append(args, filter.ConversationID)
	}

	if filter.Status != "" {
		conditions = append(conditions, `m.status = $`+fmt.Sprint(len(args)+1))
		args = append(args, filter.Status)
	}

	if filter.Direction != "" {
		conditions = append(conditions, `m.direction = $`+fmt.Sprint(len(args)+1))
		args = append(args, filter.Direction)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT m.id, m.conversation_id, m.message_id, m.direction, m.message_type,
			m.content, m.template_id, m.template_params, m.media_url, m.status,
			m.wa_status, m.sent_at, m.delivered_at, m.read_at, m.error_message, m.created_at
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		%s
		ORDER BY m.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.MessageID,
			&msg.Direction,
			&msg.MessageType,
			&msg.Content,
			&msg.TemplateID,
			&msg.TemplateParams,
			&msg.MediaURL,
			&msg.Status,
			&msg.WAStatus,
			&msg.SentAt,
			&msg.DeliveredAt,
			&msg.ReadAt,
			&msg.ErrorMessage,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}
