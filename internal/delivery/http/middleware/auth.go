package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/delivery/http/response"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/jwt"
)

const (
	CtxUserIDKey   = "current_user_id"
	CtxUserUUIDKey = "current_user_uuid"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, domain.ErrUnauth("Authorization header is required"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, domain.ErrUnauth("Authorization header must be Bearer token"))
			return
		}

		claims, err := jwt.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			response.Error(c, domain.ErrUnauth("Invalid or expired authorization token"))
			return
		}

		c.Set(CtxUserIDKey, claims.UserID)
		c.Set(CtxUserUUIDKey, claims.UserUUID)
		c.Next()
	}
}

func GetCurrentUserID(c *gin.Context) int64 {
	if val, ok := c.Get(CtxUserIDKey); ok {
		if id, ok := val.(int64); ok {
			return id
		}
	}
	return 0
}

func GetCurrentUserUUID(c *gin.Context) uuid.UUID {
	if val, ok := c.Get(CtxUserUUIDKey); ok {
		if uid, ok := val.(uuid.UUID); ok {
			return uid
		}
	}
	return uuid.Nil
}
