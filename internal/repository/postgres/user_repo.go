package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}
	db := GetDB(ctx, r.db)
	return db.Create(user).Error
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	db := GetDB(ctx, r.db)
	var user domain.User
	if err := db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.User, error) {
	db := GetDB(ctx, r.db)
	var user domain.User
	if err := db.Where("uuid = ? AND deleted_at IS NULL", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	db := GetDB(ctx, r.db)
	var user domain.User
	if err := db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
