package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wa-server/internal/models"
)

// ConversationRow extends Conversation with joined contact and phone number info.
type ConversationRow struct {
	models.Conversation
	CustomerWAID    string `json:"customer_wa_id"`
	CustomerName    string `json:"customer_name"`
	VerifiedName    string `json:"verified_name"`
	ContactPhone    string `json:"contact_phone,omitempty"`
	WaChannelID     string `json:"wa_channel_id,omitempty"`
	WaPhoneNumberID string `json:"wa_phone_number_id,omitempty"`
}

// PhoneSummaryRow holds unread count grouped by phone number.
type PhoneSummaryRow struct {
	ID              string `json:"id"`
	PhoneNumber     string `json:"phone_number"`
	PhoneNumberID   string `json:"phone_number_id"`
	DisplayName     string `json:"display_name"`
	UnreadCount     int    `json:"unread_count"`
	VerifiedName    string `json:"verified_name"`
	IsActive        bool   `json:"is_active"`
	About           string `json:"about"`
	ProfilePicURL   string `json:"profile_picture_url"`
}

// ConversationRepository implements conversation persistence for PostgreSQL.
type ConversationRepository struct {
	db *DB
}

// NewConversationRepository creates a new ConversationRepository.
func NewConversationRepository(db *DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, conv *models.Conversation) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

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

	if conv.ID == "" {
		conv.ID = generateUUID()
	}

	var companyID, contactID, assignedAgentID interface{}
	if conv.CompanyID != "" {
		companyID = conv.CompanyID
	}
	if conv.ContactID != "" {
		contactID = conv.ContactID
	}
	if conv.AssignedAgentID != "" {
		assignedAgentID = conv.AssignedAgentID
	}

	var phoneNumber, phoneNumberID interface{}
	if conv.PhoneNumber != "" {
		phoneNumber = conv.PhoneNumber
	}
	if conv.PhoneNumberID != "" {
		phoneNumberID = conv.PhoneNumberID
	}

	query := `
		INSERT INTO conversations (
			id, company_id, contact_id, assigned_agent_id, status, name,
			last_customer_message_at, last_agent_message_at, is_24h_window_active,
			unread_count, last_message_preview, phone_number, phone_number_id, started_at, closed_at, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::uuid, $14, $15, $16, $17)
	`

	_, err := r.db.ExecContext(ctx, query,
		conv.ID,
		companyID,
		contactID,
		assignedAgentID,
		conv.Status,
		conv.Name,
		conv.LastCustomerMessageAt,
		conv.LastAgentMessageAt,
		conv.Is24hWindowActive,
		conv.UnreadCount,
		conv.LastMessagePreview,
		phoneNumber,
		phoneNumberID,
		conv.StartedAt,
		conv.ClosedAt,
		conv.CreatedAt,
		conv.UpdatedAt,
	)

	return err
}

var conversationCols = `id, COALESCE(company_id::text, ''), contact_id, COALESCE(assigned_agent_id::text, ''), status,
	COALESCE(name, ''), last_customer_message_at, last_agent_message_at, is_24h_window_active,
	unread_count, COALESCE(last_message_preview, ''), COALESCE(phone_number, ''), COALESCE(phone_number_id::text, ''),
	started_at, closed_at, created_at, updated_at`

func scanConversation(scanner interface{ Scan(dest ...interface{}) error }) (models.Conversation, error) {
	var conv models.Conversation
	err := scanner.Scan(
		&conv.ID, &conv.CompanyID, &conv.ContactID, &conv.AssignedAgentID, &conv.Status,
		&conv.Name, &conv.LastCustomerMessageAt, &conv.LastAgentMessageAt, &conv.Is24hWindowActive,
		&conv.UnreadCount, &conv.LastMessagePreview, &conv.PhoneNumber, &conv.PhoneNumberID,
		&conv.StartedAt, &conv.ClosedAt, &conv.CreatedAt, &conv.UpdatedAt,
	)
	return conv, err
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE id = $1`
	conv, err := scanConversation(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) GetByIDWithCompany(ctx context.Context, id string) (*models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE id = $1`
	conv, err := scanConversation(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE phone_number = $1 ORDER BY created_at DESC LIMIT 1`
	conv, err := scanConversation(r.db.QueryRowContext(ctx, query, phoneNumber))
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) GetByContactID(ctx context.Context, companyID, contactID string) (*models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE company_id = $1 AND contact_id = $2 ORDER BY created_at DESC LIMIT 1`
	conv, err := scanConversation(r.db.QueryRowContext(ctx, query, companyID, contactID))
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) Update(ctx context.Context, conv *models.Conversation) error {
	conv.UpdatedAt = time.Now().UTC()

	var assignedAgentID interface{} = conv.AssignedAgentID
	if assignedAgentID == "" {
		assignedAgentID = nil
	}

	query := `
		UPDATE conversations
		SET assigned_agent_id = $2, status = $3, name = $4, last_customer_message_at = $5,
			last_agent_message_at = $6, is_24h_window_active = $7, unread_count = $8,
			last_message_preview = $9, closed_at = $10, updated_at = $11
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		conv.ID,
		assignedAgentID,
		conv.Status,
		conv.Name,
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
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE company_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []models.Conversation
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}

func (r *ConversationRepository) ListByAgent(ctx context.Context, agentID string) ([]models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE assigned_agent_id = $1 ORDER BY last_customer_message_at DESC`
	rows, err := r.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []models.Conversation
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}

const conversationSelectWithJoin = `
	SELECT c.id, COALESCE(c.company_id::text, ''), COALESCE(c.contact_id::text, ''), COALESCE(c.assigned_agent_id::text, ''), c.status,
		COALESCE(c.name, ''), c.last_customer_message_at, c.last_agent_message_at, c.is_24h_window_active,
		c.unread_count, COALESCE(c.last_message_preview, ''), COALESCE(c.phone_number, ''), COALESCE(c.phone_number_id::text, ''), c.started_at, c.closed_at, c.created_at, c.updated_at,
		COALESCE(ct.wa_id, ''), COALESCE(ct.name, ''), COALESCE(ct.phone_number, ''),
		COALESCE(pn.id::text, ''), COALESCE(pn.phone_number_id::text, ''), COALESCE(pn.verified_name, '')
	FROM conversations c
	LEFT JOIN contacts ct ON c.contact_id = ct.id
	LEFT JOIN phone_numbers pn ON pn.id = c.phone_number_id`

func scanConversationRow(scanner interface{ Scan(dest ...interface{}) error }) (ConversationRow, error) {
	var row ConversationRow
	err := scanner.Scan(
		&row.ID, &row.CompanyID, &row.ContactID, &row.AssignedAgentID, &row.Status,
		&row.Name, &row.LastCustomerMessageAt, &row.LastAgentMessageAt, &row.Is24hWindowActive,
		&row.UnreadCount, &row.LastMessagePreview, &row.PhoneNumber, &row.PhoneNumberID,
		&row.StartedAt, &row.ClosedAt, &row.CreatedAt, &row.UpdatedAt,
		&row.CustomerWAID, &row.CustomerName, &row.ContactPhone,
		&row.WaChannelID, &row.WaPhoneNumberID, &row.VerifiedName,
	)
	return row, err
}

// ListWithCursor returns conversations with cursor-based pagination.
// cursorID and cursorUpdatedAt are optional (empty string/zero time means start from newest).
// search filters by customer name or WA ID (empty string means no filter).
// filter can be "all", "unread", or "read".
func (r *ConversationRepository) ListWithCursor(ctx context.Context, cursorID string, cursorUpdatedAt time.Time, limit int, search, filter string) ([]ConversationRow, error) {
	if limit <= 0 {
		limit = 50
	}

	args := []interface{}{limit}
	conditions := []string{}

	if cursorID != "" && !cursorUpdatedAt.IsZero() {
		conditions = append(conditions, `(c.updated_at, c.id) < ($2, $3::uuid)`)
		args = append(args, cursorUpdatedAt, cursorID)
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf(`(ct.name ILIKE '%%' || $%d || '%%' OR ct.wa_id LIKE '%%' || $%d || '%%')`, len(args)+1, len(args)+1))
		args = append(args, search)
	}

	switch filter {
	case "unread":
		conditions = append(conditions, "c.unread_count > 0")
	case "read":
		conditions = append(conditions, "c.unread_count = 0")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := conversationSelectWithJoin + " " + whereClause + ` ORDER BY c.updated_at DESC, c.id DESC LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var result []ConversationRow
	for rows.Next() {
		row, err := scanConversationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// ListByPhoneNumberIDWithCursor filters conversations by phone_number_id with cursor pagination.
func (r *ConversationRepository) ListByPhoneNumberIDWithCursor(ctx context.Context, phoneNumberID, cursorID string, cursorUpdatedAt time.Time, limit int, search, filter string) ([]ConversationRow, error) {
	if limit <= 0 {
		limit = 50
	}

	args := []interface{}{phoneNumberID, limit}
	conditions := []string{"c.phone_number_id = $1::uuid"}

	if cursorID != "" && !cursorUpdatedAt.IsZero() {
		conditions = append(conditions, `(c.updated_at, c.id) < ($3, $4::uuid)`)
		args = append(args, cursorUpdatedAt, cursorID)
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf(`(ct.name ILIKE '%%' || $%d || '%%' OR ct.wa_id LIKE '%%' || $%d || '%%')`, len(args)+1, len(args)+1))
		args = append(args, search)
	}

	switch filter {
	case "unread":
		conditions = append(conditions, "c.unread_count > 0")
	case "read":
		conditions = append(conditions, "c.unread_count = 0")
	}

	query := conversationSelectWithJoin + " WHERE " + strings.Join(conditions, " AND ") + ` ORDER BY c.updated_at DESC, c.id DESC LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list conversations by phone: %w", err)
	}
	defer rows.Close()

	var result []ConversationRow
	for rows.Next() {
		row, err := scanConversationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetUnreadSummaryByPhoneNumber returns unread counts grouped by phone number.
func (r *ConversationRepository) GetUnreadSummaryByPhoneNumber(ctx context.Context) ([]PhoneSummaryRow, error) {
	query := `
		SELECT pn.id, pn.phone_number, pn.phone_number_id,
			COALESCE(pn.verified_name, ''), COALESCE(pn.about, ''),
			COALESCE(pn.profile_picture_url, ''),
			pn.is_active,
			COALESCE(SUM(c.unread_count), 0)::int
		FROM phone_numbers pn
		LEFT JOIN conversations c ON c.phone_number = pn.phone_number
		GROUP BY pn.id, pn.phone_number, pn.phone_number_id, pn.verified_name, pn.about, pn.profile_picture_url, pn.is_active
		ORDER BY pn.phone_number
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("unread summary: %w", err)
	}
	defer rows.Close()

	var result []PhoneSummaryRow
	for rows.Next() {
		var row PhoneSummaryRow
		if err := rows.Scan(&row.ID, &row.PhoneNumber, &row.PhoneNumberID,
			&row.VerifiedName, &row.About, &row.ProfilePicURL,
			&row.IsActive, &row.UnreadCount); err != nil {
			return nil, fmt.Errorf("scan phone summary: %w", err)
		}
		row.DisplayName = row.VerifiedName
		if row.DisplayName == "" {
			row.DisplayName = row.PhoneNumber
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *ConversationRepository) GetByPhoneNumberAndContact(ctx context.Context, phoneNumber, contactID string) (*models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE phone_number = $1 AND contact_id = $2::uuid ORDER BY created_at DESC LIMIT 1`
	conv, err := scanConversation(r.db.QueryRowContext(ctx, query, phoneNumber, contactID))
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepository) ListOpen(ctx context.Context, companyID string) ([]models.Conversation, error) {
	query := `SELECT ` + conversationCols + ` FROM conversations WHERE company_id = $1 AND status IN ('open', 'assigned', 'escalated') ORDER BY last_customer_message_at DESC`
	rows, err := r.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []models.Conversation
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}
