package http

import (
	"github.com/gin-gonic/gin"

	"github.com/MAbduhI/task-management-api/internal/delivery/http/response"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/usecase"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
}

func NewAuthHandler(authUsecase usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in usecase.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	res, err := h.authUsecase.Register(c.Request.Context(), in)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, res)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in usecase.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	res, err := h.authUsecase.Login(c.Request.Context(), in)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, res)
}
