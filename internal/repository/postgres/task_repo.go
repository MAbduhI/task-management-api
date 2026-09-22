package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type taskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) domain.TaskRepository {
	return &taskRepo{db: db}
}

func (r *taskRepo) Create(ctx context.Context, task *domain.Task) error {
	if task.UUID == uuid.Nil {
		task.UUID = uuid.New()
	}
	db := GetDB(ctx, r.db)
	return db.Create(task).Error
}

func (r *taskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	db := GetDB(ctx, r.db)
	var task domain.Task
	err := db.Preload("Creator").Preload("Assignee").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *taskRepo) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.Task, error) {
	db := GetDB(ctx, r.db)
	var task domain.Task
	err := db.Preload("Creator").Preload("Assignee").
		Where("uuid = ? AND deleted_at IS NULL", uid).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *taskRepo) Update(ctx context.Context, task *domain.Task) error {
	db := GetDB(ctx, r.db)
	return db.Save(task).Error
}

func (r *taskRepo) Delete(ctx context.Context, id int64) error {
	db := GetDB(ctx, r.db)
	now := time.Now().UTC()
	return db.Model(&domain.Task{}).Where("id = ?", id).Update("deleted_at", now).Error
}

func (r *taskRepo) List(ctx context.Context, q domain.ListTaskQuery) (*domain.TaskListResult, error) {
	db := GetDB(ctx, r.db)

	baseQuery := db.Model(&domain.Task{}).
		Where("deleted_at IS NULL").
		Where("(created_by = ? OR assignee_id = ?)", q.UserID, q.UserID)

	if q.Status != "" {
		baseQuery = baseQuery.Where("status = ?", q.Status)
	}

	if q.Search != "" {
		baseQuery = baseQuery.Where("title ILIKE ?", "%"+q.Search+"%")
	}

	var totalCount int64
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	query := baseQuery.Preload("Creator").Preload("Assignee").Order("id DESC")

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	if q.Cursor > 0 {
		query = query.Where("id < ?", q.Cursor).Limit(limit)
	} else {
		page := q.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * limit
		query = query.Offset(offset).Limit(limit)
	}

	var tasks []domain.Task
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}

	var nextCursor int64
	if len(tasks) > 0 {
		nextCursor = tasks[len(tasks)-1].ID
	}

	return &domain.TaskListResult{
		Items:      tasks,
		TotalCount: totalCount,
		NextCursor: nextCursor,
	}, nil
}
