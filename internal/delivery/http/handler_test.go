package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/jwt"
	"github.com/MAbduhI/task-management-api/internal/repository/memory"
	"github.com/MAbduhI/task-management-api/internal/repository/mock"
	"github.com/MAbduhI/task-management-api/internal/usecase"
)

func TestStructuredErrorFormat(t *testing.T) {
	jwtSecret := "test-secret"
	router := SetupRouter(RouterConfig{
		JWTSecret:   jwtSecret,
		AuthHandler: &AuthHandler{},
		TeamHandler: &TeamHandler{},
		TaskHandler: &TaskHandler{},
	})

	// 1. Request without auth header should return 401 with required error structure
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/tasks", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got: %d", w.Code)
	}

	var errResp domain.AppError
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse structured error json: %v", err)
	}

	if errResp.Status != http.StatusUnauthorized {
		t.Fatalf("expected status 401 in body, got %d", errResp.Status)
	}
	if errResp.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code 'UNAUTHORIZED', got '%s'", errResp.Code)
	}
	if errResp.Message == "" {
		t.Fatalf("expected non-empty error message")
	}
	if errResp.Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be present in error response")
	}
}

func TestHealthCheck(t *testing.T) {
	router := SetupRouter(RouterConfig{
		JWTSecret:   "secret",
		AuthHandler: &AuthHandler{},
		TeamHandler: &TeamHandler{},
		TaskHandler: &TaskHandler{},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", w.Code)
	}
}

func TestTaskCreation_WithIdempotencyHeader(t *testing.T) {
	jwtSecret := "test-secret"
	userRepo := mock.NewMockUserRepo()
	taskRepo := mock.NewMockTaskRepo()
	teamRepo := mock.NewMockTeamRepo()
	taskLogRepo := mock.NewMockTaskLogRepo()
	notifier := &mock.MockNotifier{}
	txManager := mock.NewMockTxManager(taskRepo, taskLogRepo)
	idempStore := memory.NewMemoryIdempotencyStore()

	taskUc := usecase.NewTaskUsecase(taskRepo, taskLogRepo, userRepo, teamRepo, idempStore, txManager, notifier, 24*time.Hour)
	taskHandler := NewTaskHandler(taskUc)

	router := SetupRouter(RouterConfig{
		JWTSecret:   jwtSecret,
		AuthHandler: &AuthHandler{},
		TeamHandler: &TeamHandler{},
		TaskHandler: taskHandler,
	})

	userUUID := uuid.New()
	token, _ := jwt.GenerateToken(1, userUUID, jwtSecret, time.Hour)

	idempKey := uuid.NewString()
	payload := `{"title":"Setup CI/CD","description":"GitHub Actions workflow","status":"TODO"}`

	// First call -> 201 Created
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/tasks", strings.NewReader(payload))
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("Idempotency-Key", idempKey)
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got: %d, body: %s", w1.Code, w1.Body.String())
	}

	// Second call with same idempotency key -> 201 Created from cache with header X-Idempotency-Cache: HIT
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/tasks", strings.NewReader(payload))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Idempotency-Key", idempKey)
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on duplicate, got: %d, body: %s", w2.Code, w2.Body.String())
	}
	if w2.Header().Get("X-Idempotency-Cache") != "HIT" {
		t.Fatalf("expected X-Idempotency-Cache header to be HIT")
	}

	// Verify only 1 task in repository
	if count := taskRepo.CreatedCount(); count != 1 {
		t.Fatalf("expected exactly 1 task created, found %d", count)
	}
}
