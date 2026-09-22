package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID         uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	Name         string     `gorm:"size:100;not null" json:"name"`
	Email        string     `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUUID(ctx context.Context, uid uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}
