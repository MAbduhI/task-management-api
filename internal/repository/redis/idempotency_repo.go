package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type redisIdempotencyStore struct {
	client *goredis.Client
}

func NewRedisIdempotencyStore(client *goredis.Client) domain.IdempotencyStore {
	return &redisIdempotencyStore{client: client}
}

func (s *redisIdempotencyStore) key(userID int64, idempKey string) string {
	return fmt.Sprintf("idemp:%d:%s", userID, idempKey)
}

func (s *redisIdempotencyStore) LockOrGet(ctx context.Context, idempKey string, userID int64, ttl time.Duration) (*domain.IdempotencyRecord, bool, error) {
	k := s.key(userID, idempKey)

	// Attempt atomic set if not exists for in-progress lock (30s timeout)
	ok, err := s.client.SetNX(ctx, k, string(domain.IdempotencyStatusInProgress), 30*time.Second).Result()
	if err != nil {
		return nil, false, err
	}
	if ok {
		return nil, true, nil
	}

	// Key already exists, fetch existing data
	val, err := s.client.Get(ctx, k).Result()
	if err != nil {
		if err == goredis.Nil {
			// Expired between set and get, retry lock
			return s.LockOrGet(ctx, idempKey, userID, ttl)
		}
		return nil, false, err
	}

	if val == string(domain.IdempotencyStatusInProgress) {
		return &domain.IdempotencyRecord{
			Key:    idempKey,
			UserID: userID,
			Status: domain.IdempotencyStatusInProgress,
		}, false, nil
	}

	var rec domain.IdempotencyRecord
	if err := json.Unmarshal([]byte(val), &rec); err != nil {
		return nil, false, err
	}

	return &rec, false, nil
}

func (s *redisIdempotencyStore) Resolve(ctx context.Context, idempKey string, userID int64, code int, body []byte, ttl time.Duration) error {
	k := s.key(userID, idempKey)
	now := time.Now().UTC()
	rec := domain.IdempotencyRecord{
		Key:          idempKey,
		UserID:       userID,
		Status:       domain.IdempotencyStatusResolved,
		ResponseCode: code,
		ResponseBody: string(body),
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, k, string(data), ttl).Err()
}

func (s *redisIdempotencyStore) Release(ctx context.Context, idempKey string, userID int64) error {
	k := s.key(userID, idempKey)
	return s.client.Del(ctx, k).Err()
}
