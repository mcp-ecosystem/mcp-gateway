package database

import (
	"context"

	"gorm.io/gorm"
)

// txKey is the context key used to store transactions
type txKey struct{}

// TransactionFromContext extracts a transaction from the context
func TransactionFromContext(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return nil
	}
	return tx
}

// ContextWithTransaction creates a context containing a transaction
func ContextWithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// getDBFromContext gets the DB object, using the transaction from context if available
func getDBFromContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	tx := TransactionFromContext(ctx)
	if tx != nil {
		return tx
	}
	return db.WithContext(ctx)
}
