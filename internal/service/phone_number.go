package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wa-server/internal/models"
)

type PhoneNumberRepository interface {
	Upsert(ctx context.Context, pn *models.PhoneNumber) error
	GetByID(ctx context.Context, id string) (*models.PhoneNumber, error)
	List(ctx context.Context) ([]models.PhoneNumber, error)
	AssignCompany(ctx context.Context, id, companyID string) error
	UpdateIsActive(ctx context.Context, id string, isActive bool) error
	UpdateProfile(ctx context.Context, pn *models.PhoneNumber) error
}

type ConversationRepoForPhoneNumber interface {
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Conversation, error)
}

type WhatsAppClientForPhoneNumber interface {
	GetPhoneNumbers(ctx context.Context) ([]models.WhatsAppPhoneNumber, error)
	GetBusinessProfile(ctx context.Context, phoneNumberID string) (*models.WhatsAppBusinessProfile, error)
	UpdateBusinessProfile(ctx context.Context, phoneNumberID string, profile *models.WhatsAppBusinessProfile) error
}

type PhoneNumberService struct {
	repo     PhoneNumberRepository
	convRepo ConversationRepoForPhoneNumber
	whatsapp WhatsAppClientForPhoneNumber
}

func NewPhoneNumberService(repo PhoneNumberRepository, convRepo ConversationRepoForPhoneNumber, whatsapp WhatsAppClientForPhoneNumber) *PhoneNumberService {
	return &PhoneNumberService{repo: repo, convRepo: convRepo, whatsapp: whatsapp}
}

func (s *PhoneNumberService) SyncFromMeta(ctx context.Context) (int, error) {
	numbers, err := s.whatsapp.GetPhoneNumbers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get phone numbers from Meta: %w", err)
	}

	synced := 0
	for _, n := range numbers {
		conv, _ := s.convRepo.GetByPhoneNumber(ctx, n.DisplayNumber)
		companyID := ""
		if conv != nil {
			companyID = conv.CompanyID
		}

		pn := &models.PhoneNumber{
			PhoneNumber:   n.DisplayNumber,
			PhoneNumberID: n.ID,
			VerifiedName:  n.VerifiedName,
			CompanyID:     companyID,
			IsActive:      true,
		}

		if err := s.repo.Upsert(ctx, pn); err != nil {
			slog.Error("failed to upsert phone number", "phone", n.DisplayNumber, "error", err)
			continue
		}

		profile, pErr := s.whatsapp.GetBusinessProfile(ctx, n.ID)
		if pErr != nil {
			slog.Warn("failed to fetch business profile", "phone", n.DisplayNumber, "error", pErr)
		} else {
			websitesJSON, _ := json.Marshal(profile.Websites)
			pn.About = profile.About
			pn.Address = profile.Address
			pn.Description = profile.Description
			pn.Email = profile.Email
			pn.Websites = string(websitesJSON)
			pn.Vertical = profile.Vertical
			pn.ProfilePictureURL = profile.ProfilePictureURL
			if err := s.repo.UpdateProfile(ctx, pn); err != nil {
				slog.Error("failed to update phone number profile", "phone", n.DisplayNumber, "error", err)
			}
		}

		synced++
	}

	slog.Info("phone number sync completed", "total_from_meta", len(numbers), "synced", synced)
	return synced, nil
}

func (s *PhoneNumberService) List(ctx context.Context) ([]models.PhoneNumber, error) {
	return s.repo.List(ctx)
}

func (s *PhoneNumberService) AssignToCompany(ctx context.Context, id, companyID string) (*models.PhoneNumber, error) {
	if err := s.repo.AssignCompany(ctx, id, companyID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *PhoneNumberService) GetProfile(ctx context.Context, id string) (*models.WhatsAppBusinessProfile, error) {
	pn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("phone number not found: %w", err)
	}
	return s.whatsapp.GetBusinessProfile(ctx, pn.PhoneNumberID)
}

func (s *PhoneNumberService) UpdateProfile(ctx context.Context, id string, profile *models.WhatsAppBusinessProfile) error {
	pn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("phone number not found: %w", err)
	}
	return s.whatsapp.UpdateBusinessProfile(ctx, pn.PhoneNumberID, profile)
}

func (s *PhoneNumberService) UpdateIsActive(ctx context.Context, id string, isActive bool) (*models.PhoneNumber, error) {
	if err := s.repo.UpdateIsActive(ctx, id, isActive); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}
