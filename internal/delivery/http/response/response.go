package response

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type SuccessEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Meta struct {
	Page        int   `json:"page,omitempty"`
	Limit       int   `json:"limit"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages,omitempty"`
	NextCursor  int64 `json:"next_cursor,omitempty"`
	HasNextPage bool  `json:"has_next_page"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, SuccessEnvelope{
		Success: true,
		Message: "Success",
		Data:    data,
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, SuccessEnvelope{
		Success: true,
		Message: "Created successfully",
		Data:    data,
	})
}

func List(c *gin.Context, data any, meta *Meta) {
	c.JSON(http.StatusOK, SuccessEnvelope{
		Success: true,
		Message: "Success",
		Data:    data,
		Meta:    meta,
	})
}

func Error(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		if appErr.Status >= 500 {
			slog.Error("server error",
				slog.Int("status", appErr.Status),
				slog.String("code", appErr.Code),
				slog.String("error", appErr.Error()),
				slog.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(appErr.Status, domain.AppError{
				Status:    appErr.Status,
				Code:      "INTERNAL_SERVER_ERROR",
				Message:   "Internal server error",
				Timestamp: appErr.Timestamp,
			})
			return
		}

		c.AbortWithStatusJSON(appErr.Status, appErr)
		return
	}

	// Fallback for unclassified errors
	slog.Error("unhandled server error",
		slog.String("error", err.Error()),
		slog.String("path", c.Request.URL.Path),
	)
	c.AbortWithStatusJSON(http.StatusInternalServerError, domain.AppError{
		Status:    http.StatusInternalServerError,
		Code:      "INTERNAL_SERVER_ERROR",
		Message:   "Internal server error",
		Timestamp: time.Now().UTC(),
	})
}
