package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wa-server/internal/auth"
	"github.com/wa-server/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByCompanyID(ctx context.Context, companyID string) ([]models.User, error)
	ListAll(ctx context.Context) ([]models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
}

type UserService struct {
	repo UserRepository
	jwt  *auth.JWT
}

func NewUserService(repo UserRepository, jwt *auth.JWT) *UserService {
	return &UserService{repo: repo, jwt: jwt}
}

type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *models.User `json:"user"`
}

func (s *UserService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	slog.Info("login attempt", "email", email)
	
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		slog.Error("user not found", "email", email, "error", err)
		return nil, fmt.Errorf("invalid email or password")
	}

	slog.Info("user found", "email", email, "hash", user.PasswordHash[:20], "active", user.IsActive)

	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	passwordValid := auth.CheckPassword(password, user.PasswordHash)
	slog.Info("password check", "result", passwordValid)

	if !passwordValid {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, expiresAt, err := s.jwt.GenerateToken(user.ID, user.CompanyID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

type CreateUserInput struct {
	CompanyID string          `json:"company_id"`
	Email     string          `json:"email"`
	Password  string          `json:"password"`
	Name      string          `json:"name"`
	Role      models.UserRole `json:"role"`
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*models.User, error) {
	if !input.Role.Valid() {
		return nil, fmt.Errorf("invalid role: %s", input.Role)
	}

	existing, _ := s.repo.GetByEmail(ctx, input.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:           uuid.New().String(),
		CompanyID:    input.CompanyID,
		Email:        input.Email,
		PasswordHash: hash,
		Name:         input.Name,
		Role:         string(input.Role),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) ListByCompany(ctx context.Context, companyID string) ([]models.User, error) {
	users, err := s.repo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, nil
}

func (s *UserService) ListAll(ctx context.Context) ([]models.User, error) {
	users, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, nil
}

type UpdateUserInput struct {
	ID       string
	Name     *string
	Email    *string
	Password *string
	Role     *string
	IsActive *bool
}

func (s *UserService) Update(ctx context.Context, input UpdateUserInput) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.Password != nil {
		hash, err := auth.HashPassword(*input.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.PasswordHash = hash
	}
	if input.Role != nil {
		role := models.UserRole(*input.Role)
		if !role.Valid() {
			return nil, fmt.Errorf("invalid role: %s", *input.Role)
		}
		user.Role = *input.Role
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}

	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
