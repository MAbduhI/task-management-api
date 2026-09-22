package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/sanitize"
)

type CreateTeamInput struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type AddTeamMemberInput struct {
	UserUUID uuid.UUID       `json:"user_uuid" binding:"required"`
	Role     domain.TeamRole `json:"role"`
}

type TeamResponse struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type TeamUsecase interface {
	CreateTeam(ctx context.Context, userID int64, in CreateTeamInput) (*domain.Team, error)
	AddMember(ctx context.Context, currentUserID int64, teamUUID uuid.UUID, in AddTeamMemberInput) (*domain.TeamMember, error)
	ListUserTeams(ctx context.Context, userID int64) ([]domain.Team, error)
}

type teamUsecase struct {
	teamRepo domain.TeamRepository
	userRepo domain.UserRepository
}

func NewTeamUsecase(teamRepo domain.TeamRepository, userRepo domain.UserRepository) TeamUsecase {
	return &teamUsecase{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func (u *teamUsecase) CreateTeam(ctx context.Context, userID int64, in CreateTeamInput) (*domain.Team, error) {
	name := sanitize.Text(in.Name)
	if name == "" {
		return nil, domain.ErrBadReq("VALIDATION_ERROR", "Team name cannot be empty", nil)
	}

	team := &domain.Team{
		UUID: uuid.New(),
		Name: name,
	}
	if err := u.teamRepo.Create(ctx, team); err != nil {
		return nil, domain.ErrInternal(err)
	}

	// Add creator as ADMIN of the team
	member := &domain.TeamMember{
		UUID:   uuid.New(),
		TeamID: team.ID,
		UserID: userID,
		Role:   domain.TeamRoleAdmin,
	}
	if err := u.teamRepo.AddMember(ctx, member); err != nil {
		return nil, domain.ErrInternal(err)
	}

	return team, nil
}

func (u *teamUsecase) AddMember(ctx context.Context, currentUserID int64, teamUUID uuid.UUID, in AddTeamMemberInput) (*domain.TeamMember, error) {
	team, err := u.teamRepo.GetByUUID(ctx, teamUUID)
	if err != nil {
		return nil, domain.ErrNotFoundCustom("TEAM_NOT_FOUND", "Team not found")
	}

	// Verify requester is in the team
	currentMember, err := u.teamRepo.GetMember(ctx, team.ID, currentUserID)
	if err != nil || currentMember == nil {
		return nil, domain.ErrForbid("You are not a member of this team")
	}

	targetUser, err := u.userRepo.GetByUUID(ctx, in.UserUUID)
	if err != nil {
		return nil, domain.ErrNotFoundCustom("USER_NOT_FOUND", "Target user not found")
	}

	role := in.Role
	if role == "" {
		role = domain.TeamRoleMember
	}

	member := &domain.TeamMember{
		UUID:   uuid.New(),
		TeamID: team.ID,
		UserID: targetUser.ID,
		Role:   role,
	}

	if err := u.teamRepo.AddMember(ctx, member); err != nil {
		return nil, domain.ErrInternal(err)
	}

	return member, nil
}

func (u *teamUsecase) ListUserTeams(ctx context.Context, userID int64) ([]domain.Team, error) {
	return u.teamRepo.ListUserTeams(ctx, userID)
}
