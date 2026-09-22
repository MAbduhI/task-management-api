package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusDone       TaskStatus = "DONE"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID        uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	Status      TaskStatus `gorm:"size:20;not null;default:'TODO'" json:"status"`
	CreatedBy   int64      `gorm:"not null;index:idx_tasks_creator" json:"created_by"`
	AssigneeID  *int64     `gorm:"index:idx_tasks_assignee" json:"assignee_id"`
	CreatedAt   time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`

	Creator  *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Assignee *User `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
}

type CreateTaskInput struct {
	Title       string     `json:"title" binding:"required,min=1,max=255"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
}

type UpdateTaskInput struct {
	Title       string     `json:"title" binding:"required,min=1,max=255"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status" binding:"required"`
}

type AssignTaskInput struct {
	AssigneeUUID uuid.UUID `json:"assignee_uuid" binding:"required"`
}

type ListTaskQuery struct {
	UserID int64
	Status string
	Search string
	Page   int
	Limit  int
	Cursor int64
}

type TaskResponse struct {
	UUID         uuid.UUID  `json:"uuid"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       TaskStatus `json:"status"`
	CreatorUUID  uuid.UUID  `json:"creator_uuid"`
	AssigneeUUID *uuid.UUID `json:"assignee_uuid,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type TaskListResult struct {
	Items      []Task
	TotalCount int64
	NextCursor int64
}

type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	GetByUUID(ctx context.Context, uid uuid.UUID) (*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query ListTaskQuery) (*TaskListResult, error)
}

type TaskCache interface {
	Get(ctx context.Context, taskUUID uuid.UUID) (*TaskResponse, error)
	Set(ctx context.Context, task *TaskResponse, ttl time.Duration) error
	Delete(ctx context.Context, taskUUID uuid.UUID) error
}
