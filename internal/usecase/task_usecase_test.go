package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/repository/memory"
	"github.com/MAbduhI/task-management-api/internal/repository/mock"
)

func setupTestEnvironment() (
	TaskUsecase,
	*mock.MockTaskRepo,
	*mock.MockUserRepo,
	*mock.MockTeamRepo,
	*mock.MockTaskLogRepo,
	*mock.MockNotifier,
	*mock.MockTxManager,
	*mock.MockTaskCache,
) {
	taskRepo := mock.NewMockTaskRepo()
	userRepo := mock.NewMockUserRepo()
	teamRepo := mock.NewMockTeamRepo()
	taskLogRepo := mock.NewMockTaskLogRepo()
	notifier := &mock.MockNotifier{}
	txManager := mock.NewMockTxManager(taskRepo, taskLogRepo)
	idempotencyStore := memory.NewMemoryIdempotencyStore()
	taskCache := mock.NewMockTaskCache()

	usecase := NewTaskUsecase(
		taskRepo,
		taskLogRepo,
		userRepo,
		teamRepo,
		idempotencyStore,
		taskCache,
		txManager,
		notifier,
		24*time.Hour,
	)

	return usecase, taskRepo, userRepo, teamRepo, taskLogRepo, notifier, txManager, taskCache
}

// 5.1 Race Condition - Sequential Idempotency
func TestIdempotency_Sequential(t *testing.T) {
	uc, taskRepo, userRepo, _, _, _, _, _ := setupTestEnvironment()
	ctx := context.Background()

	user := &domain.User{UUID: uuid.New(), Name: "Alice", Email: "alice@test.com"}
	_ = userRepo.Create(ctx, user)

	idempKey := uuid.NewString()
	input := domain.CreateTaskInput{
		Title:       "Deploy Application",
		Description: "Production release v1.0",
		Status:      domain.TaskStatusTodo,
	}

	// 1. First request: creates task
	resp1, cached1, err1 := uc.Create(ctx, user.ID, idempKey, input)
	if err1 != nil {
		t.Fatalf("expected request 1 to succeed, got: %v", err1)
	}
	if cached1 {
		t.Fatalf("expected request 1 to be new, got cached=true")
	}
	if resp1 == nil || resp1.UUID == uuid.Nil {
		t.Fatalf("expected valid task response")
	}

	// 2. Second request with same key: returns identical response from cache
	resp2, cached2, err2 := uc.Create(ctx, user.ID, idempKey, input)
	if err2 != nil {
		t.Fatalf("expected request 2 to succeed, got: %v", err2)
	}
	if !cached2 {
		t.Fatalf("expected request 2 to be cached, got cached=false")
	}
	if resp2.UUID != resp1.UUID {
		t.Fatalf("expected identical task UUID, got %v vs %v", resp2.UUID, resp1.UUID)
	}
	if resp2.Title != resp1.Title {
		t.Fatalf("expected identical title, got %s vs %s", resp2.Title, resp1.Title)
	}

	// 3. Verify exactly 1 task created in repository
	if count := taskRepo.CreatedCount(); count != 1 {
		t.Fatalf("expected exactly 1 task in database, found %d", count)
	}
}

// 5.1 Race Condition - Concurrent Duplicate (N goroutines with same Idempotency-Key)
func TestIdempotency_ConcurrentDuplicate(t *testing.T) {
	uc, taskRepo, userRepo, _, _, _, _, _ := setupTestEnvironment()
	ctx := context.Background()

	user := &domain.User{UUID: uuid.New(), Name: "Bob", Email: "bob@test.com"}
	_ = userRepo.Create(ctx, user)

	idempKey := uuid.NewString()
	input := domain.CreateTaskInput{
		Title:       "Process Monthly Payroll",
		Description: "Finance batch execution",
		Status:      domain.TaskStatusTodo,
	}

	const concurrency = 30
	var wg sync.WaitGroup
	wg.Add(concurrency)

	type testResult struct {
		resp   *domain.TaskResponse
		cached bool
		err    error
	}

	results := make([]testResult, concurrency)

	// Launch concurrent requests simultaneously
	for i := 0; i < concurrency; i++ {
		idx := i
		go func() {
			defer wg.Done()
			r, c, err := uc.Create(ctx, user.ID, idempKey, input)
			results[idx] = testResult{resp: r, cached: c, err: err}
		}()
	}

	wg.Wait()

	// Assert: Exactly ONE task created in database
	createdCount := taskRepo.CreatedCount()
	if createdCount != 1 {
		t.Fatalf("RACE CONDITION DETECTED! Expected exactly 1 task created in DB, got: %d", createdCount)
	}

	// Assert: All successful responses returned the same task UUID
	var initialUUID uuid.UUID
	successCount := 0
	conflictCount := 0

	for _, res := range results {
		if res.err == nil && res.resp != nil {
			successCount++
			if initialUUID == uuid.Nil {
				initialUUID = res.resp.UUID
			} else if res.resp.UUID != initialUUID {
				t.Fatalf("Inconsistent task UUID returned across duplicate requests: %v vs %v", res.resp.UUID, initialUUID)
			}
		} else if res.err != nil {
			var appErr *domain.AppError
			if errors.As(res.err, &appErr) && appErr.Code == "CONCURRENT_REQUEST" {
				conflictCount++
			} else {
				t.Fatalf("Unexpected error under concurrency: %v", res.err)
			}
		}
	}

	if successCount == 0 {
		t.Fatalf("expected at least 1 successful request, got 0")
	}

	t.Logf("Concurrency test passed: %d goroutines, %d succeeded, %d locked in-flight, exactly %d created in DB",
		concurrency, successCount, conflictCount, createdCount)
}

// 3. Database Transaction & Integrity - Rollback on step failure
func TestAssign_NotifierFailureDoesNotRollback(t *testing.T) {
	uc, taskRepo, userRepo, teamRepo, taskLogRepo, notifier, _, _ := setupTestEnvironment()
	ctx := context.Background()

	assigner := &domain.User{UUID: uuid.New(), Name: "Alice", Email: "alice@team.com"}
	_ = userRepo.Create(ctx, assigner)

	assignee := &domain.User{UUID: uuid.New(), Name: "Charlie", Email: "charlie@team.com"}
	_ = userRepo.Create(ctx, assignee)

	team := &domain.Team{UUID: uuid.New(), Name: "Core Dev"}
	_ = teamRepo.Create(ctx, team)
	_ = teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: assigner.ID, Role: domain.TeamRoleAdmin})
	_ = teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: assignee.ID, Role: domain.TeamRoleMember})

	task := &domain.Task{
		UUID:      uuid.New(),
		Title:     "Refactor Auth Middleware",
		Status:    domain.TaskStatusTodo,
		CreatedBy: assigner.ID,
	}
	_ = taskRepo.Create(ctx, task)

	notifier.ShouldErr = true

	resp, err := uc.Assign(ctx, assigner.ID, task.UUID, assignee.UUID)
	if err != nil {
		t.Fatalf("expected assignment to succeed despite notifier error, got: %v", err)
	}

	if resp.AssigneeUUID == nil || *resp.AssigneeUUID != assignee.UUID {
		t.Fatalf("expected assignee UUID %v, got %v", assignee.UUID, resp.AssigneeUUID)
	}

	taskAfter, _ := taskRepo.GetByUUID(ctx, task.UUID)
	if taskAfter.AssigneeID == nil || *taskAfter.AssigneeID != assignee.ID {
		t.Fatalf("expected task assignee_id to be %d", assignee.ID)
	}

	if taskLogRepo.Count() != 1 {
		t.Fatalf("expected 1 task log (audit committed), got %d", taskLogRepo.Count())
	}
}

func TestAssign_Success(t *testing.T) {
	uc, taskRepo, userRepo, teamRepo, taskLogRepo, notifier, _, _ := setupTestEnvironment()
	ctx := context.Background()

	assigner := &domain.User{UUID: uuid.New(), Name: "Alice", Email: "alice@team.com"}
	_ = userRepo.Create(ctx, assigner)

	assignee := &domain.User{UUID: uuid.New(), Name: "Charlie", Email: "charlie@team.com"}
	_ = userRepo.Create(ctx, assignee)

	team := &domain.Team{UUID: uuid.New(), Name: "Core Dev"}
	_ = teamRepo.Create(ctx, team)
	_ = teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: assigner.ID, Role: domain.TeamRoleAdmin})
	_ = teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: assignee.ID, Role: domain.TeamRoleMember})

	task := &domain.Task{
		UUID:      uuid.New(),
		Title:     "Write Documentation",
		Status:    domain.TaskStatusTodo,
		CreatedBy: assigner.ID,
	}
	_ = taskRepo.Create(ctx, task)

	resp, err := uc.Assign(ctx, assigner.ID, task.UUID, assignee.UUID)
	if err != nil {
		t.Fatalf("expected assign to succeed, got: %v", err)
	}
	if resp.AssigneeUUID == nil || *resp.AssigneeUUID != assignee.UUID {
		t.Fatalf("expected assignee UUID %v, got %v", assignee.UUID, resp.AssigneeUUID)
	}

	taskAfter, _ := taskRepo.GetByUUID(ctx, task.UUID)
	if taskAfter.AssigneeID == nil || *taskAfter.AssigneeID != assignee.ID {
		t.Fatalf("expected task assignee_id to be %d", assignee.ID)
	}

	if taskLogRepo.Count() != 1 {
		t.Fatalf("expected 1 task log, got %d", taskLogRepo.Count())
	}

	if notifier.SentCount != 1 {
		t.Fatalf("expected 1 notification sent, got %d", notifier.SentCount)
	}
}

func TestAssign_DifferentTeamRejected(t *testing.T) {
	uc, taskRepo, userRepo, teamRepo, taskLogRepo, _, _, _ := setupTestEnvironment()
	ctx := context.Background()

	user1 := &domain.User{UUID: uuid.New(), Name: "User 1", Email: "user1@a.com"}
	_ = userRepo.Create(ctx, user1)

	user2 := &domain.User{UUID: uuid.New(), Name: "User 2", Email: "user2@b.com"}
	_ = userRepo.Create(ctx, user2)

	team1 := &domain.Team{UUID: uuid.New(), Name: "Team 1"}
	_ = teamRepo.Create(ctx, team1)
	_ = teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team1.ID, UserID: user1.ID})

	team2 := &domain.Team{UUID: uuid.New(), Name: "Team 2"}
	_ = teamRepo.Create(ctx, team2)
	_ = teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team2.ID, UserID: user2.ID})

	task := &domain.Task{
		UUID:      uuid.New(),
		Title:     "Private Task",
		Status:    domain.TaskStatusTodo,
		CreatedBy: user1.ID,
	}
	_ = taskRepo.Create(ctx, task)

	// Attempt assignment across different teams
	_, err := uc.Assign(ctx, user1.ID, task.UUID, user2.UUID)
	if err == nil {
		t.Fatalf("expected assignment across different teams to fail, got nil")
	}

	var appErr *domain.AppError
	if !errors.As(err, &appErr) || appErr.Code != "DIFFERENT_TEAM" {
		t.Fatalf("expected DIFFERENT_TEAM error, got: %v", err)
	}

	if taskLogRepo.Count() != 0 {
		t.Fatalf("expected 0 task logs, got %d", taskLogRepo.Count())
	}
}

func TestUpdate_InvalidatesCache(t *testing.T) {
	uc, taskRepo, userRepo, _, _, _, _, taskCache := setupTestEnvironment()
	ctx := context.Background()

	user := &domain.User{UUID: uuid.New(), Name: "Alice", Email: "alice@test.com"}
	_ = userRepo.Create(ctx, user)

	task := &domain.Task{
		UUID:      uuid.New(),
		Title:     "Old Title",
		Status:    domain.TaskStatusTodo,
		CreatedBy: user.ID,
	}
	_ = taskRepo.Create(ctx, task)

	// Pre-fill cache
	_ = taskCache.Set(ctx, &domain.TaskResponse{UUID: task.UUID, Title: "Old Title"}, time.Hour)

	// Update task
	_, err := uc.Update(ctx, user.ID, task.UUID, domain.UpdateTaskInput{
		Title:  "New Title",
		Status: domain.TaskStatusInProgress,
	})
	if err != nil {
		t.Fatalf("expected update to succeed, got: %v", err)
	}

	// Verify cache was invalidated
	cached, _ := taskCache.Get(ctx, task.UUID)
	if cached != nil {
		t.Fatalf("expected cache to be invalidated on update, but found cached entry")
	}
	if taskCache.DeletedCount() != 1 {
		t.Fatalf("expected 1 cache delete call, got %d", taskCache.DeletedCount())
	}
}

func TestDelete_InvalidatesCache(t *testing.T) {
	uc, taskRepo, userRepo, _, _, _, _, taskCache := setupTestEnvironment()
	ctx := context.Background()

	user := &domain.User{UUID: uuid.New(), Name: "Alice", Email: "alice@test.com"}
	_ = userRepo.Create(ctx, user)

	task := &domain.Task{
		UUID:      uuid.New(),
		Title:     "Task to Delete",
		Status:    domain.TaskStatusTodo,
		CreatedBy: user.ID,
	}
	_ = taskRepo.Create(ctx, task)

	// Pre-fill cache
	_ = taskCache.Set(ctx, &domain.TaskResponse{UUID: task.UUID, Title: "Task to Delete"}, time.Hour)

	// Delete task
	err := uc.Delete(ctx, user.ID, task.UUID)
	if err != nil {
		t.Fatalf("expected delete to succeed, got: %v", err)
	}

	// Verify cache was invalidated
	cached, _ := taskCache.Get(ctx, task.UUID)
	if cached != nil {
		t.Fatalf("expected cache to be invalidated on delete, but found cached entry")
	}
	if taskCache.DeletedCount() != 1 {
		t.Fatalf("expected 1 cache delete call, got %d", taskCache.DeletedCount())
	}
}
