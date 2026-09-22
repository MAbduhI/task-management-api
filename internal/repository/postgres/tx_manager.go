package postgres

import (
	"context"

	"gorm.io/gorm"
)

type contextKey string

const txKey contextKey = "gorm_tx"

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type GormTxManager struct {
	db *gorm.DB
}

func NewGormTxManager(db *gorm.DB) *GormTxManager {
	return &GormTxManager{db: db}
}

func (m *GormTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// If already in a transaction, reuse it
	if _, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return fn(ctx)
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey, tx)
		return fn(txCtx)
	})
}

// GetDB retrieves the ambient transaction from context, or falls back to db
func GetDB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return fallback.WithContext(ctx)
}
