package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/whatsapp"
)

type TemplateService struct {
	repo  models.TemplateRepository
	wacli *whatsapp.Client
}

func NewTemplateService(repo models.TemplateRepository, wacli *whatsapp.Client) *TemplateService {
	return &TemplateService{repo: repo, wacli: wacli}
}

func (s *TemplateService) Create(ctx context.Context, tmpl *models.Template) error {
	return s.repo.Create(ctx, tmpl)
}

func (s *TemplateService) GetByID(ctx context.Context, id string) (*models.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TemplateService) GetByName(ctx context.Context, name string) ([]models.Template, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *TemplateService) List(ctx context.Context, limit, offset int) ([]models.Template, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *TemplateService) Delete(ctx context.Context, id string) error {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tmpl.MetaName != "" {
		if err := s.wacli.DeleteTemplate(ctx, tmpl.MetaName); err != nil {
			slog.Error("failed to delete template from Meta", "error", err, "meta_name", tmpl.MetaName)
		}
	}

	return s.repo.Delete(ctx, id)
}

func (s *TemplateService) Update(ctx context.Context, tmpl *models.Template) error {
	tmpl.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, tmpl)
}

func (s *TemplateService) CreateAndSync(ctx context.Context, tmpl *models.Template) error {
	if tmpl.Name == "" {
		return fmt.Errorf("template name is required")
	}

	metaName := toMetaName(tmpl.Name)

	components := &whatsapp.TemplateComponents{
		HeaderType:    tmpl.HeaderType,
		HeaderContent: tmpl.HeaderContent,
		BodyContent:   tmpl.Content,
		FooterText:    tmpl.FooterText,
		ButtonsJSON:   tmpl.Buttons,
	}

	metaID, metaStatus, err := s.wacli.CreateTemplate(ctx, metaName, tmpl.Language, tmpl.Category, components)
	if err != nil {
		return fmt.Errorf("failed to create template in Meta: %w", err)
	}

	tmpl.MetaName = metaName
	tmpl.WATemplateID = metaID
	tmpl.MetaStatus = metaStatus
	tmpl.IsVerified = metaStatus == "APPROVED"

	return s.repo.Create(ctx, tmpl)
}

func (s *TemplateService) SyncAll(ctx context.Context) (int, error) {
	metaTemplates, err := s.wacli.GetTemplates(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch templates from Meta: %w", err)
	}

	synced := 0
	for _, mt := range metaTemplates {
		existing, err := s.repo.GetByMetaNameAndLanguage(ctx, mt.Name, mt.Language)
		if err != nil {
			tmpl := &models.Template{
				WATemplateID: mt.ID,
				MetaName:     mt.Name,
				Name:         mt.Name,
				Language:     mt.Language,
				Category:     mt.Category,
				IsVerified:   mt.Status == "APPROVED",
				MetaStatus:   mt.Status,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}
			if err := s.repo.Create(ctx, tmpl); err != nil {
				slog.Error("failed to create template from Meta", "error", err, "meta_name", mt.Name)
				continue
			}
			synced++
		} else {
			existing.WATemplateID = mt.ID
			existing.MetaStatus = mt.Status
			existing.IsVerified = mt.Status == "APPROVED"
			existing.Category = mt.Category
			existing.UpdatedAt = time.Now().UTC()
			if err := s.repo.Update(ctx, existing); err != nil {
				slog.Error("failed to update template from Meta", "error", err, "id", existing.ID)
				continue
			}
			synced++
		}
	}

	return synced, nil
}

func (s *TemplateService) SyncPendingStatus(ctx context.Context) (int, error) {
	templates, err := s.repo.GetByMetaStatus(ctx, "PENDING")
	if err != nil {
		return 0, fmt.Errorf("failed to list pending templates: %w", err)
	}

	updated := 0
	for _, tmpl := range templates {

		metaTmpl, err := s.wacli.GetTemplateByName(ctx, tmpl.MetaName)
		if err != nil {
			slog.Error("failed to check template status in Meta", "error", err, "meta_name", tmpl.MetaName)
			continue
		}
		if metaTmpl == nil {
			continue
		}

		if metaTmpl.Status != tmpl.MetaStatus {
			isVerified := metaTmpl.Status == "APPROVED"
			if err := s.repo.UpdateMetaStatus(ctx, tmpl.ID, metaTmpl.Status, isVerified); err != nil {
				slog.Error("failed to update meta status", "error", err, "id", tmpl.ID)
				continue
			}
			updated++
		}
	}

	return updated, nil
}

func toMetaName(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
