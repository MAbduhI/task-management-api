package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TaskLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	TaskID      int64     `gorm:"not null;index:idx_task_logs_task" json:"task_id"`
	PerformedBy int64     `gorm:"not null" json:"performed_by"`
	Action      string    `gorm:"size:50;not null" json:"action"`
	Details     string    `gorm:"type:text;not null" json:"details"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime" json:"created_at"`

	Task        *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Performer   *User `gorm:"foreignKey:PerformedBy" json:"performer,omitempty"`
}

type TaskLogRepository interface {
	Create(ctx context.Context, log *TaskLog) error
	ListByTaskID(ctx context.Context, taskID int64) ([]TaskLog, error)
}

type TaskLogResponse struct {
	UUID        uuid.UUID `json:"uuid"`
	Action      string    `json:"action"`
	Details     string    `json:"details"`
	PerformedBy uuid.UUID `json:"performed_by"`
	CreatedAt   time.Time `json:"created_at"`
}
