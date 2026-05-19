package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wa-server/internal/models"
)

var ErrTemplateNotFound = errors.New("template not found")

type TemplateRepo struct {
	db *DB
}

func NewTemplateRepo(db *DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

func (r *TemplateRepo) Create(ctx context.Context, tmpl *models.Template) error {
	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}
	if tmpl.CreatedAt.IsZero() {
		tmpl.CreatedAt = time.Now().UTC()
	}
	if tmpl.UpdatedAt.IsZero() {
		tmpl.UpdatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO templates (id, wa_template_id, meta_name, name, language, category, content,
			header_type, header_content, body_components, footer_text, buttons,
			is_verified, meta_status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.WATemplateID, tmpl.MetaName, tmpl.Name, tmpl.Language, tmpl.Category,
		tmpl.Content, nullStr(tmpl.HeaderType), nullStr(tmpl.HeaderContent),
		nullJSON(tmpl.BodyComponents), nullStr(tmpl.FooterText),
		nullJSON(tmpl.Buttons), tmpl.IsVerified, nullStr(tmpl.MetaStatus),
		nullStr(tmpl.CreatedBy), tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}
	return nil
}

var templateCols = `id, wa_template_id, COALESCE(meta_name, ''), name, language, category, content,
	COALESCE(header_type, ''), COALESCE(header_content, ''), COALESCE(body_components::TEXT, ''), COALESCE(footer_text, ''),
	COALESCE(buttons::TEXT, ''), is_verified, COALESCE(meta_status, ''), COALESCE(created_by::TEXT, ''), created_at, updated_at`

func scanTemplate(row interface{ Scan(dest ...any) error }) (*models.Template, error) {
	var t models.Template
	err := row.Scan(
		&t.ID, &t.WATemplateID, &t.MetaName, &t.Name, &t.Language, &t.Category,
		&t.Content, &t.HeaderType, &t.HeaderContent, &t.BodyComponents,
		&t.FooterText, &t.Buttons, &t.IsVerified, &t.MetaStatus,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepo) GetByID(ctx context.Context, id string) (*models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates WHERE id = $1`
	t, err := scanTemplate(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return nil, ErrTemplateNotFound
	}
	return t, nil
}

func (r *TemplateRepo) GetByWATemplateID(ctx context.Context, waID string) (*models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates WHERE wa_template_id = $1`
	t, err := scanTemplate(r.db.QueryRowContext(ctx, query, waID))
	if err != nil {
		return nil, ErrTemplateNotFound
	}
	return t, nil
}

func (r *TemplateRepo) GetByName(ctx context.Context, name string) ([]models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates WHERE name = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepo) GetVerified(ctx context.Context) ([]models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates WHERE is_verified = true ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepo) Update(ctx context.Context, tmpl *models.Template) error {
	tmpl.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE templates
		SET wa_template_id = $2, meta_name = $3, name = $4, language = $5, category = $6, content = $7,
			header_type = $8, header_content = $9, body_components = $10, footer_text = $11,
			buttons = $12, is_verified = $13, meta_status = $14, updated_at = $15
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.WATemplateID, tmpl.MetaName, tmpl.Name, tmpl.Language, tmpl.Category,
		tmpl.Content, nullStr(tmpl.HeaderType), nullStr(tmpl.HeaderContent),
		nullJSON(tmpl.BodyComponents), nullStr(tmpl.FooterText),
		nullJSON(tmpl.Buttons), tmpl.IsVerified, nullStr(tmpl.MetaStatus),
		tmpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (r *TemplateRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM templates WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (r *TemplateRepo) GetByMetaNameAndLanguage(ctx context.Context, metaName, language string) (*models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates WHERE meta_name = $1 AND language = $2`
	t, err := scanTemplate(r.db.QueryRowContext(ctx, query, metaName, language))
	if err != nil {
		return nil, ErrTemplateNotFound
	}
	return t, nil
}

func (r *TemplateRepo) GetByMetaStatus(ctx context.Context, status string) ([]models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates WHERE meta_status = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepo) UpdateMetaStatus(ctx context.Context, id, metaStatus string, isVerified bool) error {
	query := `UPDATE templates SET meta_status = $2, is_verified = $3, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, metaStatus, isVerified)
	if err != nil {
		return fmt.Errorf("failed to update meta status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (r *TemplateRepo) List(ctx context.Context, limit, offset int) ([]models.Template, error) {
	query := `SELECT ` + templateCols + ` FROM templates ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}
