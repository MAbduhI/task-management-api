package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/MAbduhI/task-management-api/internal/delivery/http/response"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/migrations"
)

type SystemHandler struct {
	db       *gorm.DB
	adminKey string
}

func NewSystemHandler(db *gorm.DB, adminKey string) *SystemHandler {
	return &SystemHandler{
		db:       db,
		adminKey: adminKey,
	}
}

func (h *SystemHandler) verifyAdmin(c *gin.Context) bool {
	key := c.GetHeader("X-Admin-Key")
	if key == "" || key != h.adminKey {
		response.Error(c, domain.ErrForbid("Invalid or missing X-Admin-Key header"))
		return false
	}
	return true
}

func (h *SystemHandler) executeSQL(c *gin.Context, filename, actionName string) {
	if !h.verifyAdmin(c) {
		return
	}

	content, err := migrations.FS.ReadFile(filename)
	if err != nil {
		slog.Error("failed to read migration file", slog.String("file", filename), slog.String("error", err.Error()))
		response.Error(c, domain.ErrInternal(err))
		return
	}

	if err := h.db.Exec(string(content)).Error; err != nil {
		slog.Error("failed to execute sql", slog.String("file", filename), slog.String("error", err.Error()))
		response.Error(c, domain.ErrInternal(err))
		return
	}

	response.OK(c, gin.H{
		"action":  actionName,
		"file":    filename,
		"status":  "success",
		"message": actionName + " executed successfully",
	})
}

func (h *SystemHandler) MigrateUp(c *gin.Context) {
	h.executeSQL(c, "000001_init_schema.up.sql", "MIGRATE_UP")
}

func (h *SystemHandler) MigrateDown(c *gin.Context) {
	h.executeSQL(c, "000001_init_schema.down.sql", "MIGRATE_DOWN")
}

func (h *SystemHandler) SeedUp(c *gin.Context) {
	h.executeSQL(c, "000002_seed_data.up.sql", "SEED_UP")
}

func (h *SystemHandler) SeedDown(c *gin.Context) {
	h.executeSQL(c, "000002_seed_data.down.sql", "SEED_DOWN")
}
