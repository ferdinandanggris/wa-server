package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wa-server/internal/models"
)

type PhoneNumberRepository interface {
	Upsert(ctx context.Context, pn *models.PhoneNumber) error
	List(ctx context.Context) ([]models.PhoneNumber, error)
}

type ConversationRepoForPhoneNumber interface {
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Conversation, error)
}

type WhatsAppClientForPhoneNumber interface {
	GetPhoneNumbers(ctx context.Context) ([]models.WhatsAppPhoneNumber, error)
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
			CompanyID:     companyID,
			IsActive:      true,
		}

		if err := s.repo.Upsert(ctx, pn); err != nil {
			slog.Error("failed to upsert phone number", "phone", n.DisplayNumber, "error", err)
			continue
		}
		synced++
	}

	slog.Info("phone number sync completed", "total_from_meta", len(numbers), "synced", synced)
	return synced, nil
}

func (s *PhoneNumberService) List(ctx context.Context) ([]models.PhoneNumber, error) {
	return s.repo.List(ctx)
}
