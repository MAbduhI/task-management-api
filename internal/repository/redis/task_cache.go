package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

type redisTaskCache struct {
	client *goredis.Client
}

func NewRedisTaskCache(client *goredis.Client) domain.TaskCache {
	return &redisTaskCache{client: client}
}

func (c *redisTaskCache) key(taskUUID uuid.UUID) string {
	return fmt.Sprintf("task:%s", taskUUID.String())
}

func (c *redisTaskCache) Get(ctx context.Context, taskUUID uuid.UUID) (*domain.TaskResponse, error) {
	val, err := c.client.Get(ctx, c.key(taskUUID)).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var task domain.TaskResponse
	if err := json.Unmarshal([]byte(val), &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *redisTaskCache) Set(ctx context.Context, task *domain.TaskResponse, ttl time.Duration) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(task.UUID), string(data), ttl).Err()
}

func (c *redisTaskCache) Delete(ctx context.Context, taskUUID uuid.UUID) error {
	return c.client.Del(ctx, c.key(taskUUID)).Err()
}
