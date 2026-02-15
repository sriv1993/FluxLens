// Package factory opens in-memory or Postgres audit stores without an import cycle.
package factory

import (
	"context"
	"fmt"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/auditlog/postgres"
)

// Open returns a durable Postgres store when dsn is non-empty, otherwise an in-memory chain.
func Open(ctx context.Context, dsn, tenantID string) (auditlog.Store, func() error, error) {
	if dsn == "" {
		return auditlog.NewChain(), func() error { return nil }, nil
	}
	st, err := postgres.Open(ctx, dsn, tenantID)
	if err != nil {
		return nil, nil, err
	}
	return st, st.Close, nil
}

// BackendLabel describes the active store implementation.
func BackendLabel(store auditlog.Store) string {
	if _, ok := store.(*postgres.Store); ok {
		return "postgres"
	}
	return "memory"
}

// MustOpen panics on error (tests).
func MustOpen(ctx context.Context, dsn, tenantID string) (auditlog.Store, func() error) {
	st, closeFn, err := Open(ctx, dsn, tenantID)
	if err != nil {
		panic(fmt.Sprintf("auditlog factory: %v", err))
	}
	return st, closeFn
}
