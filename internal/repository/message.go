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

// MessageRepository implements models.MessageRepository for PostgreSQL.
type MessageRepository struct {
	db *DB
}

func nullJSON(s string) interface{} {
	if s == "" || s == "null" {
		return nil
	}
	return s
}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository(db *DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, msg *models.Message) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

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

	var idempotencyKey interface{}
	if msg.IdempotencyKey != "" {
		idempotencyKey = msg.IdempotencyKey
	}

	query := `
		INSERT INTO messages (
			id, conversation_id, message_id, direction, message_type,
			content, template_id, template_params, media_url, status,
			wa_status, sent_at, delivered_at, read_at, error_message, created_at, idempotency_key
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
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
		idempotencyKey,
	)

	if err != nil {
		slog.Error("DB insert error", "error", err, "templateID", templateID)
	}

	return err
}

func scanMessage(row interface{ Scan(dest ...any) error }) (*models.Message, error) {
	var msg models.Message
	err := row.Scan(
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
		&msg.IdempotencyKey,
	)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *MessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			COALESCE(content, ''), COALESCE(template_id::TEXT, ''), COALESCE(template_params::TEXT, ''),
			COALESCE(media_url, ''), status, COALESCE(wa_status, ''),
			sent_at, delivered_at, read_at, COALESCE(error_message, ''), created_at,
			COALESCE(idempotency_key, '')
		FROM messages WHERE id = $1
	`

	return scanMessage(r.db.QueryRowContext(ctx, query, id))
}

func (r *MessageRepository) GetByMessageID(ctx context.Context, messageID string) (*models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			COALESCE(content, ''), COALESCE(template_id::TEXT, ''), COALESCE(template_params::TEXT, ''),
			COALESCE(media_url, ''), status, COALESCE(wa_status, ''),
			sent_at, delivered_at, read_at, COALESCE(error_message, ''), created_at,
			COALESCE(idempotency_key, '')
		FROM messages WHERE message_id = $1
	`

	return scanMessage(r.db.QueryRowContext(ctx, query, messageID))
}

func (r *MessageRepository) GetByIdempotencyKey(ctx context.Context, key string) (*models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			COALESCE(content, ''), COALESCE(template_id::TEXT, ''), COALESCE(template_params::TEXT, ''),
			COALESCE(media_url, ''), status, COALESCE(wa_status, ''),
			sent_at, delivered_at, read_at, COALESCE(error_message, ''), created_at,
			COALESCE(idempotency_key, '')
		FROM messages WHERE idempotency_key = $1
	`

	return scanMessage(r.db.QueryRowContext(ctx, query, key))
}

func (r *MessageRepository) GetByConversationID(ctx context.Context, convID string, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT id, conversation_id, message_id, direction, message_type,
			COALESCE(content, ''), COALESCE(template_id::TEXT, ''), COALESCE(template_params::TEXT, ''),
			COALESCE(media_url, ''), status, COALESCE(wa_status, ''),
			sent_at, delivered_at, read_at, COALESCE(error_message, ''), created_at,
			COALESCE(idempotency_key, '')
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
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}

	return messages, rows.Err()
}

func (r *MessageRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE messages SET status = $1 WHERE id = $2`
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

func (r *MessageRepository) UpdateDeliveryStatus(ctx context.Context, id, status string, timestamp time.Time) error {
	var query string
	switch status {
	case "sent":
		query = `UPDATE messages SET status = $1, sent_at = $2 WHERE id = $3`
	case "delivered":
		query = `UPDATE messages SET status = $1, delivered_at = $2 WHERE id = $3`
	case "read":
		query = `UPDATE messages SET status = $1, read_at = $2 WHERE id = $3`
	default:
		query = `UPDATE messages SET status = $1 WHERE id = $3`
	}
	result, err := r.db.ExecContext(ctx, query, status, timestamp, id)
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

func (r *MessageRepository) SetFailed(ctx context.Context, id, errMsg string) error {
	query := `UPDATE messages SET status = 'failed', error_message = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, errMsg, id)
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

// MessageFilter holds optional filters for the List query.
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
		args = append(args, filter.CompanyID)
		conditions = append(conditions, `c.company_id = $`+fmt.Sprint(len(args)))
	}

	if filter.ConversationID != "" {
		args = append(args, filter.ConversationID)
		conditions = append(conditions, `m.conversation_id = $`+fmt.Sprint(len(args)))
	}

	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, `m.status = $`+fmt.Sprint(len(args)))
	}

	if filter.Direction != "" {
		args = append(args, filter.Direction)
		conditions = append(conditions, `m.direction = $`+fmt.Sprint(len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	args = append(args, limit, offset)
	limitIdx := len(args) - 1
	offsetIdx := len(args)

	query := fmt.Sprintf(`
		SELECT m.id, m.conversation_id, m.message_id, m.direction, m.message_type,
			COALESCE(m.content, ''), COALESCE(m.template_id::TEXT, ''), COALESCE(m.template_params::TEXT, ''),
			COALESCE(m.media_url, ''), m.status, COALESCE(m.wa_status, ''),
			m.sent_at, m.delivered_at, m.read_at, COALESCE(m.error_message, ''), m.created_at,
			COALESCE(m.idempotency_key, '')
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		%s
		ORDER BY m.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}

	return messages, rows.Err()
}
