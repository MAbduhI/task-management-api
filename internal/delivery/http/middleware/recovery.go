package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		reqID, _ := c.Get(RequestIDKey)
		slog.Error("panic recovered",
			slog.Any("request_id", reqID),
			slog.Any("panic", recovered),
			slog.String("path", c.Request.URL.Path),
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, domain.AppError{
			Status:    http.StatusInternalServerError,
			Code:      "INTERNAL_SERVER_ERROR",
			Message:   "Internal server error",
			Timestamp: time.Now().UTC(),
		})
	})
}
