package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type teamRepo struct {
	db *gorm.DB
}

func NewTeamRepo(db *gorm.DB) domain.TeamRepository {
	return &teamRepo{db: db}
}

func (r *teamRepo) Create(ctx context.Context, team *domain.Team) error {
	if team.UUID == uuid.Nil {
		team.UUID = uuid.New()
	}
	db := GetDB(ctx, r.db)
	return db.Create(team).Error
}

func (r *teamRepo) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	db := GetDB(ctx, r.db)
	var team domain.Team
	if err := db.Where("id = ?", id).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (r *teamRepo) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.Team, error) {
	db := GetDB(ctx, r.db)
	var team domain.Team
	if err := db.Where("uuid = ?", uid).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (r *teamRepo) AddMember(ctx context.Context, member *domain.TeamMember) error {
	if member.UUID == uuid.Nil {
		member.UUID = uuid.New()
	}
	db := GetDB(ctx, r.db)
	return db.Create(member).Error
}

func (r *teamRepo) GetMember(ctx context.Context, teamID, userID int64) (*domain.TeamMember, error) {
	db := GetDB(ctx, r.db)
	var member domain.TeamMember
	if err := db.Where("team_id = ? AND user_id = ?", teamID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (r *teamRepo) AreUsersInSameTeam(ctx context.Context, userID1, userID2 int64) (bool, error) {
	db := GetDB(ctx, r.db)
	var count int64
	err := db.Table("team_members tm1").
		Joins("JOIN team_members tm2 ON tm1.team_id = tm2.team_id").
		Where("tm1.user_id = ? AND tm2.user_id = ?", userID1, userID2).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teamRepo) ListUserTeams(ctx context.Context, userID int64) ([]domain.Team, error) {
	db := GetDB(ctx, r.db)
	var teams []domain.Team
	err := db.Table("teams").
		Joins("JOIN team_members tm ON tm.team_id = teams.id").
		Where("tm.user_id = ?", userID).
		Find(&teams).Error
	return teams, err
}
