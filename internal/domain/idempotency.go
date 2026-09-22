package domain

import (
	"context"
	"time"
)

type IdempotencyStatus string

const (
	IdempotencyStatusInProgress IdempotencyStatus = "IN_PROGRESS"
	IdempotencyStatusResolved   IdempotencyStatus = "RESOLVED"
)

type IdempotencyRecord struct {
	ID           int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Key          string            `gorm:"size:64;not null;index:idx_user_key,unique" json:"key"`
	UserID       int64             `gorm:"not null;index:idx_user_key,unique" json:"user_id"`
	Status       IdempotencyStatus `gorm:"size:20;not null" json:"status"`
	ResponseCode int               `json:"response_code"`
	ResponseBody string            `gorm:"type:text" json:"response_body"`
	CreatedAt    time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
	ExpiresAt    time.Time         `gorm:"not null;index" json:"expires_at"`
}

type IdempotencyStore interface {
	// LockOrGet attempts to acquire an in-progress lock for the key.
	// If the key is already resolved, it returns the cached record and acquired=false.
	// If the key is currently in-progress, it returns the record and acquired=false.
	// If the key is new, it sets status to IN_PROGRESS and returns acquired=true.
	LockOrGet(ctx context.Context, key string, userID int64, ttl time.Duration) (*IdempotencyRecord, bool, error)

	// Resolve stores the completed response and marks the key RESOLVED.
	Resolve(ctx context.Context, key string, userID int64, code int, body []byte, ttl time.Duration) error

	// Release removes an in-progress lock (used when execution fails before resolving).
	Release(ctx context.Context, key string, userID int64) error
}
