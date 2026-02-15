// Command chain-verifier loads the audit log from Postgres (or verifies an
// empty chain) and exits 0 when the hash chain is intact, 1 otherwise.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/sriharshav1/fluxlens/internal/auditlog/factory"
)

func main() {
	dsn := flag.String("postgres-dsn", os.Getenv("FLUXLENS_POSTGRES_DSN"), "Postgres DSN (required)")
	tenant := flag.String("tenant", "default", "Tenant id for fluxlens_audit_log")
	flag.Parse()
	if *dsn == "" {
		log.Fatal("postgres-dsn or FLUXLENS_POSTGRES_DSN required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, closeFn, err := factory.Open(ctx, *dsn, *tenant)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer closeFn()
	ok, err := store.Verify()
	if err != nil {
		log.Fatalf("verify error: %v", err)
	}
	if !ok {
		log.Fatalf("AUDIT CHAIN INVALID (records=%d head=%s)", store.Len(), store.HeadHash())
	}
	log.Printf("audit chain valid: records=%d head=%s backend=%s", store.Len(), store.HeadHash(), factory.BackendLabel(store))
}
