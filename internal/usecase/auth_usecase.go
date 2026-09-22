package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/jwt"
	"github.com/MAbduhI/task-management-api/internal/pkg/password"
	"github.com/MAbduhI/task-management-api/internal/pkg/sanitize"
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	UUID      uuid.UUID `json:"uuid"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthUsecase interface {
	Register(ctx context.Context, in RegisterInput) (*AuthResponse, error)
	Login(ctx context.Context, in LoginInput) (*AuthResponse, error)
}

type authUsecase struct {
	userRepo       domain.UserRepository
	jwtSecret      string
	jwtExpiryHours int
}

func NewAuthUsecase(userRepo domain.UserRepository, jwtSecret string, jwtExpiryHours int) AuthUsecase {
	return &authUsecase{
		userRepo:       userRepo,
		jwtSecret:      jwtSecret,
		jwtExpiryHours: jwtExpiryHours,
	}
}

func (u *authUsecase) Register(ctx context.Context, in RegisterInput) (*AuthResponse, error) {
	email := sanitize.Email(in.Email)
	name := sanitize.Text(in.Name)

	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, domain.ErrConflictCustom("EMAIL_ALREADY_EXISTS", "Email is already registered")
	}

	hashed, err := password.Hash(in.Password)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	user := &domain.User{
		UUID:         uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: hashed,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, domain.ErrInternal(err)
	}

	expiry := time.Duration(u.jwtExpiryHours) * time.Hour
	token, err := jwt.GenerateToken(user.ID, user.UUID, u.jwtSecret, expiry)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			UUID:      user.UUID,
			Name:      user.Name,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (u *authUsecase) Login(ctx context.Context, in LoginInput) (*AuthResponse, error) {
	email := sanitize.Email(in.Email)
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrUnauth("Invalid email or password")
	}

	if !password.Check(in.Password, user.PasswordHash) {
		return nil, domain.ErrUnauth("Invalid email or password")
	}

	expiry := time.Duration(u.jwtExpiryHours) * time.Hour
	token, err := jwt.GenerateToken(user.ID, user.UUID, u.jwtSecret, expiry)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			UUID:      user.UUID,
			Name:      user.Name,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}
