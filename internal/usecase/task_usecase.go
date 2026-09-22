package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/notification"
	"github.com/MAbduhI/task-management-api/internal/pkg/sanitize"
	"github.com/MAbduhI/task-management-api/internal/repository/postgres"
)

type TaskUsecase interface {
	Create(ctx context.Context, currentUserID int64, idempotencyKey string, in domain.CreateTaskInput) (*domain.TaskResponse, bool, error)
	List(ctx context.Context, query domain.ListTaskQuery) (*domain.TaskListResult, error)
	GetByUUID(ctx context.Context, currentUserID int64, taskUUID uuid.UUID) (*domain.TaskResponse, error)
	Update(ctx context.Context, currentUserID int64, taskUUID uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error)
	Delete(ctx context.Context, currentUserID int64, taskUUID uuid.UUID) error
	Assign(ctx context.Context, currentUserID int64, taskUUID uuid.UUID, assigneeUUID uuid.UUID) (*domain.TaskResponse, error)
}

type taskUsecase struct {
	taskRepo         domain.TaskRepository
	taskLogRepo      domain.TaskLogRepository
	userRepo         domain.UserRepository
	teamRepo         domain.TeamRepository
	idempotencyStore domain.IdempotencyStore
	txManager        postgres.TxManager
	notifier         notification.Notifier
	idempotencyTTL   time.Duration
}

func NewTaskUsecase(
	taskRepo domain.TaskRepository,
	taskLogRepo domain.TaskLogRepository,
	userRepo domain.UserRepository,
	teamRepo domain.TeamRepository,
	idempotencyStore domain.IdempotencyStore,
	txManager postgres.TxManager,
	notifier notification.Notifier,
	idempotencyTTL time.Duration,
) TaskUsecase {
	if idempotencyTTL <= 0 {
		idempotencyTTL = 24 * time.Hour
	}
	return &taskUsecase{
		taskRepo:         taskRepo,
		taskLogRepo:      taskLogRepo,
		userRepo:         userRepo,
		teamRepo:         teamRepo,
		idempotencyStore: idempotencyStore,
		txManager:        txManager,
		notifier:         notifier,
		idempotencyTTL:   idempotencyTTL,
	}
}

func (u *taskUsecase) Create(ctx context.Context, currentUserID int64, idempotencyKey string, in domain.CreateTaskInput) (*domain.TaskResponse, bool, error) {
	// 1. Idempotency handling
	if idempotencyKey != "" {
		if _, err := uuid.Parse(idempotencyKey); err != nil {
			return nil, false, domain.ErrBadReq("INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be a valid UUID", err)
		}

		if u.idempotencyStore != nil {
			rec, acquired, err := u.idempotencyStore.LockOrGet(ctx, idempotencyKey, currentUserID, u.idempotencyTTL)
			if err != nil {
				return nil, false, domain.ErrInternal(err)
			}
			if !acquired && rec != nil {
				if rec.Status == domain.IdempotencyStatusResolved {
					var cached domain.TaskResponse
					if err := json.Unmarshal([]byte(rec.ResponseBody), &cached); err == nil {
						return &cached, true, nil
					}
				}
				// In progress - concurrent duplicate
				return nil, false, domain.ErrConflictCustom("CONCURRENT_REQUEST", "A request with this idempotency key is already in progress")
			}
		}
	}

	// 2. Sanitize & Validate input
	title := sanitize.Text(in.Title)
	if title == "" {
		if idempotencyKey != "" && u.idempotencyStore != nil {
			_ = u.idempotencyStore.Release(ctx, idempotencyKey, currentUserID)
		}
		return nil, false, domain.ErrBadReq("VALIDATION_ERROR", "Task title cannot be empty", nil)
	}
	description := sanitize.Text(in.Description)

	status := in.Status
	if status == "" {
		status = domain.TaskStatusTodo
	} else if !status.IsValid() {
		if idempotencyKey != "" && u.idempotencyStore != nil {
			_ = u.idempotencyStore.Release(ctx, idempotencyKey, currentUserID)
		}
		return nil, false, domain.ErrBadReq("INVALID_STATUS", "Invalid task status. Allowed: TODO, IN_PROGRESS, DONE", nil)
	}

	// 3. Persist task
	task := &domain.Task{
		UUID:        uuid.New(),
		Title:       title,
		Description: description,
		Status:      status,
		CreatedBy:   currentUserID,
	}

	if err := u.taskRepo.Create(ctx, task); err != nil {
		if idempotencyKey != "" && u.idempotencyStore != nil {
			_ = u.idempotencyStore.Release(ctx, idempotencyKey, currentUserID)
		}
		return nil, false, domain.ErrInternal(err)
	}

	creator, _ := u.userRepo.GetByID(ctx, currentUserID)
	creatorUUID := uuid.Nil
	if creator != nil {
		creatorUUID = creator.UUID
	}

	resp := &domain.TaskResponse{
		UUID:        task.UUID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatorUUID: creatorUUID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	// 4. Resolve idempotency lock
	if idempotencyKey != "" && u.idempotencyStore != nil {
		bodyBytes, err := json.Marshal(resp)
		if err == nil {
			_ = u.idempotencyStore.Resolve(ctx, idempotencyKey, currentUserID, http.StatusCreated, bodyBytes, u.idempotencyTTL)
		}
	}

	return resp, false, nil
}

func (u *taskUsecase) List(ctx context.Context, query domain.ListTaskQuery) (*domain.TaskListResult, error) {
	query.Search = sanitize.Text(query.Search)
	query.Status = sanitize.Text(query.Status)
	return u.taskRepo.List(ctx, query)
}

func (u *taskUsecase) GetByUUID(ctx context.Context, currentUserID int64, taskUUID uuid.UUID) (*domain.TaskResponse, error) {
	task, err := u.taskRepo.GetByUUID(ctx, taskUUID)
	if err != nil {
		return nil, domain.ErrNotFoundCustom("TASK_NOT_FOUND", "Task not found")
	}

	if !u.hasTaskAccess(task, currentUserID) {
		return nil, domain.ErrForbid("You do not have access to this task")
	}

	return u.mapTaskToResponse(task), nil
}

func (u *taskUsecase) Update(ctx context.Context, currentUserID int64, taskUUID uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error) {
	task, err := u.taskRepo.GetByUUID(ctx, taskUUID)
	if err != nil {
		return nil, domain.ErrNotFoundCustom("TASK_NOT_FOUND", "Task not found")
	}

	if !u.hasTaskAccess(task, currentUserID) {
		return nil, domain.ErrForbid("You do not have permission to update this task")
	}

	if !in.Status.IsValid() {
		return nil, domain.ErrBadReq("INVALID_STATUS", "Invalid task status. Allowed: TODO, IN_PROGRESS, DONE", nil)
	}

	title := sanitize.Text(in.Title)
	if title == "" {
		return nil, domain.ErrBadReq("VALIDATION_ERROR", "Task title cannot be empty", nil)
	}

	task.Title = title
	task.Description = sanitize.Text(in.Description)
	task.Status = in.Status

	if err := u.taskRepo.Update(ctx, task); err != nil {
		return nil, domain.ErrInternal(err)
	}

	return u.mapTaskToResponse(task), nil
}

func (u *taskUsecase) Delete(ctx context.Context, currentUserID int64, taskUUID uuid.UUID) error {
	task, err := u.taskRepo.GetByUUID(ctx, taskUUID)
	if err != nil {
		return domain.ErrNotFoundCustom("TASK_NOT_FOUND", "Task not found")
	}

	if task.CreatedBy != currentUserID {
		return domain.ErrForbid("Only the task creator can delete this task")
	}

	return u.taskRepo.Delete(ctx, task.ID)
}

func (u *taskUsecase) Assign(ctx context.Context, currentUserID int64, taskUUID uuid.UUID, assigneeUUID uuid.UUID) (*domain.TaskResponse, error) {
	var updatedTask *domain.TaskResponse

	err := u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Fetch task
		task, err := u.taskRepo.GetByUUID(txCtx, taskUUID)
		if err != nil {
			return domain.ErrNotFoundCustom("TASK_NOT_FOUND", "Task not found")
		}

		if !u.hasTaskAccess(task, currentUserID) {
			return domain.ErrForbid("You do not have permission to assign this task")
		}

		// 2. Fetch target assignee
		assignee, err := u.userRepo.GetByUUID(txCtx, assigneeUUID)
		if err != nil {
			return domain.ErrNotFoundCustom("ASSIGNEE_NOT_FOUND", "Assignee user not found")
		}

		// 3. Verify both users share at least one team
		sameTeam, err := u.teamRepo.AreUsersInSameTeam(txCtx, currentUserID, assignee.ID)
		if err != nil {
			return domain.ErrInternal(err)
		}
		if !sameTeam {
			return domain.ErrBadReq("DIFFERENT_TEAM", "Assignee must belong to the same team as the assigner", nil)
		}

		// 4. Update task assignee
		task.AssigneeID = &assignee.ID
		if err := u.taskRepo.Update(txCtx, task); err != nil {
			return domain.ErrInternal(err)
		}

		// 5. Audit log
		logDetails, _ := json.Marshal(map[string]any{
			"assignee_uuid": assignee.UUID,
			"assignee_id":   assignee.ID,
		})
		logEntry := &domain.TaskLog{
			UUID:        uuid.New(),
			TaskID:      task.ID,
			PerformedBy: currentUserID,
			Action:      "ASSIGNED",
			Details:     string(logDetails),
		}
		if err := u.taskLogRepo.Create(txCtx, logEntry); err != nil {
			return domain.ErrInternal(err)
		}

		// 6. Notify assignee (failure causes rollback)
		if u.notifier != nil {
			msg := fmt.Sprintf("You have been assigned to task: %s", task.Title)
			if err := u.notifier.Send(txCtx, assignee.ID, "Task Assignment", msg); err != nil {
				return domain.ErrInternal(fmt.Errorf("failed to send notification: %w", err))
			}
		}

		task.Assignee = assignee
		updatedTask = u.mapTaskToResponse(task)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedTask, nil
}

func (u *taskUsecase) hasTaskAccess(task *domain.Task, userID int64) bool {
	if task.CreatedBy == userID {
		return true
	}
	if task.AssigneeID != nil && *task.AssigneeID == userID {
		return true
	}
	return false
}

func (u *taskUsecase) mapTaskToResponse(task *domain.Task) *domain.TaskResponse {
	var creatorUUID uuid.UUID
	if task.Creator != nil {
		creatorUUID = task.Creator.UUID
	}
	var assigneeUUID *uuid.UUID
	if task.Assignee != nil {
		assigneeUUID = &task.Assignee.UUID
	}

	return &domain.TaskResponse{
		UUID:         task.UUID,
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		CreatorUUID:  creatorUUID,
		AssigneeUUID: assigneeUUID,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}
