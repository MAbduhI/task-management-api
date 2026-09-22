package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type taskLogRepo struct {
	db *gorm.DB
}

func NewTaskLogRepo(db *gorm.DB) domain.TaskLogRepository {
	return &taskLogRepo{db: db}
}

func (r *taskLogRepo) Create(ctx context.Context, log *domain.TaskLog) error {
	if log.UUID == uuid.Nil {
		log.UUID = uuid.New()
	}
	db := GetDB(ctx, r.db)
	return db.Create(log).Error
}

func (r *taskLogRepo) ListByTaskID(ctx context.Context, taskID int64) ([]domain.TaskLog, error) {
	db := GetDB(ctx, r.db)
	var logs []domain.TaskLog
	err := db.Preload("Performer").
		Where("task_id = ?", taskID).
		Order("id ASC").
		Find(&logs).Error
	return logs, err
}
