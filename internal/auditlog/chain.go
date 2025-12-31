// Package auditlog implements an append-only, hash-chained audit log.
//
// Every decision and operator action in FluxLens is recorded as a Record
// in the audit log. Records are linked by SHA-256 hash chain: each Record
// includes the hash of its predecessor, and its own hash is computed over
// the canonical serialization of its fields plus the predecessor hash.
// Any tampering with a historical record breaks the chain at and after
// the modified record, detectable by Verify().
//
// This in-memory implementation is suitable for unit testing, local
// development, and small embedded deployments. Production deployments
// should plug in a durable backend (Postgres-backed implementation lives
// under internal/auditlog/postgres in a later phase).
package auditlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Record is one entry in the audit chain.
type Record struct {
	Sequence  uint64          `json:"sequence"`
	Timestamp time.Time       `json:"timestamp"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	PrevHash  string          `json:"prev_hash"`
	Hash      string          `json:"hash"`
}

// Chain is an in-memory append-only audit chain.
type Chain struct {
	mu       sync.Mutex
	records  []Record
	headHash string
	nextSeq  uint64
}

// NewChain returns a fresh empty chain.
func NewChain() *Chain {
	return &Chain{}
}

// Append serializes payload, computes the next chain hash, and appends.
// Returns the persisted Record on success.
func (c *Chain) Append(kind string, payload any) (Record, error) {
	if kind == "" {
		return Record{}, errors.New("auditlog: kind required")
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return Record{}, fmt.Errorf("auditlog: marshal payload: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	rec := Record{
		Sequence:  c.nextSeq,
		Timestamp: time.Now().UTC(),
		Kind:      kind,
		Payload:   pb,
		PrevHash:  c.headHash,
	}
	h, err := computeHash(&rec)
	if err != nil {
		return Record{}, err
	}
	rec.Hash = h
	c.records = append(c.records, rec)
	c.headHash = h
	c.nextSeq++
	return rec, nil
}

// Verify recomputes the chain hashes and returns (true, nil) iff the
// chain is intact. On the first mismatch, returns (false, nil). Errors
// indicate serialization failure, not tampering.
func (c *Chain) Verify() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := ""
	for i := range c.records {
		expected, err := computeHashWithPrev(&c.records[i], prev)
		if err != nil {
			return false, err
		}
		if c.records[i].Hash != expected {
			return false, nil
		}
		if c.records[i].PrevHash != prev {
			return false, nil
		}
		prev = c.records[i].Hash
	}
	return true, nil
}

// Len returns the number of records currently in the chain.
func (c *Chain) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.records)
}

// HeadHash returns the hash of the most recent record (or "" if empty).
func (c *Chain) HeadHash() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.headHash
}

// Snapshot returns a copy of all records (safe for caller mutation).
func (c *Chain) Snapshot() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Record, len(c.records))
	copy(out, c.records)
	return out
}

// computeHash computes a Record's hash using its current PrevHash field.
func computeHash(r *Record) (string, error) {
	return computeHashWithPrev(r, r.PrevHash)
}

// computeHashWithPrev computes the canonical hash for r assuming the
// given prev. Independent of r.PrevHash so the verifier can confirm the
// chain shape end-to-end.
func computeHashWithPrev(r *Record, prev string) (string, error) {
	body, err := canonicalBytes(r, prev)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalBytes returns the deterministic byte representation of the
// fields used to compute the hash. The Hash field itself is excluded.
func canonicalBytes(r *Record, prev string) ([]byte, error) {
	type hashable struct {
		Sequence  uint64          `json:"sequence"`
		Timestamp string          `json:"timestamp"`
		Kind      string          `json:"kind"`
		Payload   json.RawMessage `json:"payload"`
		PrevHash  string          `json:"prev_hash"`
	}
	return json.Marshal(hashable{
		Sequence:  r.Sequence,
		Timestamp: r.Timestamp.UTC().Format(time.RFC3339Nano),
		Kind:      r.Kind,
		Payload:   r.Payload,
		PrevHash:  prev,
	})
}
