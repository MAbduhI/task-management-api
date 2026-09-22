package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TeamRole string

const (
	TeamRoleAdmin  TeamRole = "ADMIN"
	TeamRoleMember TeamRole = "MEMBER"
)

type Team struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

type TeamMember struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	TeamID    int64     `gorm:"not null;index:idx_team_user,unique" json:"team_id"`
	UserID    int64     `gorm:"not null;index:idx_team_user,unique;index" json:"user_id"`
	Role      TeamRole  `gorm:"size:20;not null;default:'MEMBER'" json:"role"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`

	Team *Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	GetByID(ctx context.Context, id int64) (*Team, error)
	GetByUUID(ctx context.Context, uid uuid.UUID) (*Team, error)
	AddMember(ctx context.Context, member *TeamMember) error
	GetMember(ctx context.Context, teamID, userID int64) (*TeamMember, error)
	AreUsersInSameTeam(ctx context.Context, userID1, userID2 int64) (bool, error)
	ListUserTeams(ctx context.Context, userID int64) ([]Team, error)
}
