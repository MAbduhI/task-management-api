package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/delivery/http/middleware"
	"github.com/MAbduhI/task-management-api/internal/delivery/http/response"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/usecase"
)

type TeamHandler struct {
	teamUsecase usecase.TeamUsecase
}

func NewTeamHandler(teamUsecase usecase.TeamUsecase) *TeamHandler {
	return &TeamHandler{teamUsecase: teamUsecase}
}

func (h *TeamHandler) Create(c *gin.Context) {
	var in usecase.CreateTeamInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	userID := middleware.GetCurrentUserID(c)
	team, err := h.teamUsecase.CreateTeam(c.Request.Context(), userID, in)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, team)
}

func (h *TeamHandler) AddMember(c *gin.Context) {
	teamUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.Error(c, domain.ErrBadReq("INVALID_UUID", "Invalid team UUID", err))
		return
	}

	var in usecase.AddTeamMemberInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	currentUserID := middleware.GetCurrentUserID(c)
	member, err := h.teamUsecase.AddMember(c.Request.Context(), currentUserID, teamUUID, in)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, member)
}

func (h *TeamHandler) ListMyTeams(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	teams, err := h.teamUsecase.ListUserTeams(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, teams)
}
