package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wa-server/internal/models"
)

type CompanyRepository interface {
	Create(ctx context.Context, company *models.Company) error
	GetByID(ctx context.Context, id string) (*models.Company, error)
	GetByCode(ctx context.Context, code string) (*models.Company, error)
	Update(ctx context.Context, company *models.Company) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]models.Company, error)
}

type CompanyService struct {
	repo CompanyRepository
}

func NewCompanyService(repo CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
}

type CreateCompanyInput struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	PhoneNumber string `json:"phone_number"`
	Address     string `json:"address"`
	QuotaLimit  int    `json:"quota_limit"`
}

func (s *CompanyService) Create(ctx context.Context, input CreateCompanyInput) (*models.Company, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	now := time.Now().UTC()
	company := &models.Company{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Code:        input.Code,
		PhoneNumber: toNullString(input.PhoneNumber),
		Address:     toNullString(input.Address),
		IsActive:    true,
		QuotaLimit:  input.QuotaLimit,
		QuotaUsed:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if company.QuotaLimit <= 0 {
		company.QuotaLimit = 50000
	}

	if err := s.repo.Create(ctx, company); err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}

	return company, nil
}

func (s *CompanyService) GetByID(ctx context.Context, id string) (*models.Company, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CompanyService) List(ctx context.Context, limit, offset int) ([]models.Company, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, limit, offset)
}

type UpdateCompanyInput struct {
	ID          string
	Name        *string `json:"name"`
	PhoneNumber *string `json:"phone_number"`
	Address     *string `json:"address"`
	IsActive    *bool   `json:"is_active"`
	QuotaLimit  *int    `json:"quota_limit"`
}

func (s *CompanyService) Update(ctx context.Context, input UpdateCompanyInput) (*models.Company, error) {
	company, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		company.Name = *input.Name
	}
	if input.PhoneNumber != nil {
		company.PhoneNumber = toNullStringPtr(input.PhoneNumber)
	}
	if input.Address != nil {
		company.Address = toNullStringPtr(input.Address)
	}
	if input.IsActive != nil {
		company.IsActive = *input.IsActive
	}
	if input.QuotaLimit != nil {
		company.QuotaLimit = *input.QuotaLimit
	}

	if err := s.repo.Update(ctx, company); err != nil {
		return nil, fmt.Errorf("update company: %w", err)
	}

	return company, nil
}

func (s *CompanyService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullStringPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return toNullString(*s)
}
