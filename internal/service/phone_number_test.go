package service

import (
	"context"
	"errors"
	"testing"

	"github.com/wa-server/internal/models"
)

type mockPhoneNumberRepo struct {
	upsertFunc func(ctx context.Context, pn *models.PhoneNumber) error
	listFunc   func(ctx context.Context) ([]models.PhoneNumber, error)
}

func (m *mockPhoneNumberRepo) Upsert(ctx context.Context, pn *models.PhoneNumber) error {
	return m.upsertFunc(ctx, pn)
}
func (m *mockPhoneNumberRepo) List(ctx context.Context) ([]models.PhoneNumber, error) {
	return m.listFunc(ctx)
}

type mockConvRepoForPhone struct {
	getByPhoneNumberFunc func(ctx context.Context, phoneNumber string) (*models.Conversation, error)
}

func (m *mockConvRepoForPhone) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Conversation, error) {
	return m.getByPhoneNumberFunc(ctx, phoneNumber)
}

type mockWhatsappPhone struct {
	getPhoneNumbersFunc func(ctx context.Context) ([]models.WhatsAppPhoneNumber, error)
}

func (m *mockWhatsappPhone) GetPhoneNumbers(ctx context.Context) ([]models.WhatsAppPhoneNumber, error) {
	return m.getPhoneNumbersFunc(ctx)
}

func TestPhoneNumberService_SyncFromMeta_Success(t *testing.T) {
	numbers := []models.WhatsAppPhoneNumber{
		{ID: "pn1", DisplayNumber: "+62811", VerifiedName: "Phone 1"},
		{ID: "pn2", DisplayNumber: "+62822", VerifiedName: "Phone 2"},
	}

	upserted := make(map[string]string)
	repo := &mockPhoneNumberRepo{
		upsertFunc: func(ctx context.Context, pn *models.PhoneNumber) error {
			upserted[pn.PhoneNumber] = pn.PhoneNumberID
			return nil
		},
	}
	convRepo := &mockConvRepoForPhone{
		getByPhoneNumberFunc: func(ctx context.Context, phoneNumber string) (*models.Conversation, error) {
			return nil, errors.New("not found")
		},
	}
	w := &mockWhatsappPhone{
		getPhoneNumbersFunc: func(ctx context.Context) ([]models.WhatsAppPhoneNumber, error) {
			return numbers, nil
		},
	}

	svc := NewPhoneNumberService(repo, convRepo, w)
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if synced != 2 {
		t.Fatalf("synced = %d, want 2", synced)
	}
	if upserted["+62811"] != "pn1" || upserted["+62822"] != "pn2" {
		t.Fatalf("upserted = %v", upserted)
	}
}

func TestPhoneNumberService_SyncFromMeta_APIError(t *testing.T) {
	w := &mockWhatsappPhone{
		getPhoneNumbersFunc: func(ctx context.Context) ([]models.WhatsAppPhoneNumber, error) {
			return nil, errors.New("meta API error")
		},
	}
	svc := NewPhoneNumberService(&mockPhoneNumberRepo{}, &mockConvRepoForPhone{}, w)
	_, err := svc.SyncFromMeta(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPhoneNumberService_SyncFromMeta_PartialUpsertError(t *testing.T) {
	numbers := []models.WhatsAppPhoneNumber{
		{ID: "pn1", DisplayNumber: "+62811", VerifiedName: "Phone 1"},
		{ID: "pn2", DisplayNumber: "+62822", VerifiedName: "Phone 2"},
	}

	attempts := 0
	repo := &mockPhoneNumberRepo{
		upsertFunc: func(ctx context.Context, pn *models.PhoneNumber) error {
			attempts++
			if pn.PhoneNumber == "+62822" {
				return errors.New("db error")
			}
			return nil
		},
	}
	w := &mockWhatsappPhone{
		getPhoneNumbersFunc: func(ctx context.Context) ([]models.WhatsAppPhoneNumber, error) {
			return numbers, nil
		},
	}

	svc := NewPhoneNumberService(repo, &mockConvRepoForPhone{
		getByPhoneNumberFunc: func(ctx context.Context, phoneNumber string) (*models.Conversation, error) {
			return nil, errors.New("not found")
		},
	}, w)
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal("should not return error on partial failure")
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2 (both should be attempted)", attempts)
	}
}

func TestPhoneNumberService_SyncFromMeta_MapsCompany(t *testing.T) {
	numbers := []models.WhatsAppPhoneNumber{
		{ID: "pn1", DisplayNumber: "+62811", VerifiedName: "Phone 1"},
	}

	repo := &mockPhoneNumberRepo{
		upsertFunc: func(ctx context.Context, pn *models.PhoneNumber) error {
			if pn.CompanyID != "c1" {
				t.Fatalf("CompanyID = %q, want c1", pn.CompanyID)
			}
			return nil
		},
	}
	convRepo := &mockConvRepoForPhone{
		getByPhoneNumberFunc: func(ctx context.Context, phoneNumber string) (*models.Conversation, error) {
			return &models.Conversation{CompanyID: "c1"}, nil
		},
	}
	w := &mockWhatsappPhone{
		getPhoneNumbersFunc: func(ctx context.Context) ([]models.WhatsAppPhoneNumber, error) {
			return numbers, nil
		},
	}

	svc := NewPhoneNumberService(repo, convRepo, w)
	synced, err := svc.SyncFromMeta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
}

func TestPhoneNumberService_List(t *testing.T) {
	expected := []models.PhoneNumber{
		{PhoneNumber: "+62811", PhoneNumberID: "pn1", IsActive: true},
	}
	repo := &mockPhoneNumberRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return expected, nil
		},
	}
	svc := NewPhoneNumberService(repo, &mockConvRepoForPhone{}, &mockWhatsappPhone{})
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PhoneNumber != "+62811" {
		t.Fatalf("got %+v", got)
	}
}

func TestPhoneNumberService_List_Error(t *testing.T) {
	repo := &mockPhoneNumberRepo{
		listFunc: func(ctx context.Context) ([]models.PhoneNumber, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewPhoneNumberService(repo, &mockConvRepoForPhone{}, &mockWhatsappPhone{})
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
