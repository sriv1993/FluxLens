package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
)

// Store persists the hash chain in fluxlens_audit_log (see deploy/postgres-init).
type Store struct {
	db       *sql.DB
	tenantID string
	mu       sync.Mutex
	headHash string
	nextSeq  uint64
}

// Open connects to Postgres and loads chain head state for tenantID.
func Open(ctx context.Context, dsn, tenantID string) (*Store, error) {
	if tenantID == "" {
		tenantID = "default"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("auditlog/postgres: open: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("auditlog/postgres: ping: %w", err)
	}
	s := &Store{db: db, tenantID: tenantID}
	if err := s.loadHead(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) loadHead(ctx context.Context) error {
	var seq sql.NullInt64
	var hash sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT sequence, hash
		FROM fluxlens_audit_log
		WHERE tenant_id = $1
		ORDER BY sequence DESC
		LIMIT 1`, s.tenantID).Scan(&seq, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("auditlog/postgres: load head: %w", err)
	}
	if seq.Valid {
		s.nextSeq = uint64(seq.Int64) + 1
		s.headHash = hash.String
	}
	return nil
}

// Close releases the database pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// Append inserts the next chained record.
func (s *Store) Append(kind string, payload any) (auditlog.Record, error) {
	if kind == "" {
		return auditlog.Record{}, errors.New("auditlog: kind required")
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return auditlog.Record{}, fmt.Errorf("auditlog: marshal payload: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec := auditlog.Record{
		Sequence:  s.nextSeq,
		Timestamp: time.Now().UTC(),
		Kind:      kind,
		Payload:   pb,
		PrevHash:  s.headHash,
	}
	h, err := auditlog.ComputeHash(&rec)
	if err != nil {
		return auditlog.Record{}, err
	}
	rec.Hash = h

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO fluxlens_audit_log
			(sequence, timestamp_utc, kind, payload, prev_hash, hash, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rec.Sequence, rec.Timestamp, rec.Kind, rec.Payload, rec.PrevHash, rec.Hash, s.tenantID)
	if err != nil {
		return auditlog.Record{}, fmt.Errorf("auditlog/postgres: insert: %w", err)
	}
	s.headHash = h
	s.nextSeq++
	return rec, nil
}

// Verify reloads the tenant chain and checks hashes.
func (s *Store) Verify() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence, timestamp_utc, kind, payload, prev_hash, hash
		FROM fluxlens_audit_log
		WHERE tenant_id = $1
		ORDER BY sequence ASC`, s.tenantID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var records []auditlog.Record
	for rows.Next() {
		var r auditlog.Record
		if err := rows.Scan(&r.Sequence, &r.Timestamp, &r.Kind, &r.Payload, &r.PrevHash, &r.Hash); err != nil {
			return false, err
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return auditlog.VerifyRecords(records)
}

// Len returns row count for this tenant.
func (s *Store) Len() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fluxlens_audit_log WHERE tenant_id = $1`, s.tenantID).Scan(&n)
	return n
}

// HeadHash returns the cached head hash (updated on Append).
func (s *Store) HeadHash() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.headHash
}

// Snapshot loads all records for the tenant.
func (s *Store) Snapshot() []auditlog.Record {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence, timestamp_utc, kind, payload, prev_hash, hash
		FROM fluxlens_audit_log
		WHERE tenant_id = $1
		ORDER BY sequence ASC`, s.tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []auditlog.Record
	for rows.Next() {
		var r auditlog.Record
		if err := rows.Scan(&r.Sequence, &r.Timestamp, &r.Kind, &r.Payload, &r.PrevHash, &r.Hash); err != nil {
			return out
		}
		out = append(out, r)
	}
	return out
}

var _ auditlog.Store = (*Store)(nil)
