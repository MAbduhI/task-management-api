package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type memoryIdempotencyStore struct {
	mu    sync.Mutex
	items map[string]*domain.IdempotencyRecord
}

func NewMemoryIdempotencyStore() domain.IdempotencyStore {
	return &memoryIdempotencyStore{
		items: make(map[string]*domain.IdempotencyRecord),
	}
}

func (s *memoryIdempotencyStore) key(userID int64, idempKey string) string {
	return fmt.Sprintf("%d:%s", userID, idempKey)
}

func (s *memoryIdempotencyStore) LockOrGet(ctx context.Context, idempKey string, userID int64, ttl time.Duration) (*domain.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := s.key(userID, idempKey)
	now := time.Now().UTC()

	if rec, exists := s.items[k]; exists {
		if now.Before(rec.ExpiresAt) {
			// Return a copy so caller modifications don't mutate store
			copied := *rec
			return &copied, false, nil
		}
		// Expired
		delete(s.items, k)
	}

	// New lock
	record := &domain.IdempotencyRecord{
		Key:       idempKey,
		UserID:    userID,
		Status:    domain.IdempotencyStatusInProgress,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	s.items[k] = record
	return nil, true, nil
}

func (s *memoryIdempotencyStore) Resolve(ctx context.Context, idempKey string, userID int64, code int, body []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := s.key(userID, idempKey)
	now := time.Now().UTC()
	s.items[k] = &domain.IdempotencyRecord{
		Key:          idempKey,
		UserID:       userID,
		Status:       domain.IdempotencyStatusResolved,
		ResponseCode: code,
		ResponseBody: string(body),
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
	}
	return nil
}

func (s *memoryIdempotencyStore) Release(ctx context.Context, idempKey string, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, s.key(userID, idempKey))
	return nil
}
