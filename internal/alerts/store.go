package alerts

import (
	"sync"
	"time"
)

// Store is an in-memory ring buffer of operator alerts. Phase 2 persists to
// Postgres and fans out to paging/webhooks (see roadmap).
type Store struct {
	mu       sync.Mutex
	buf      []Alert
	max      int
	dedupe   time.Duration
	lastRule map[string]time.Time
}

// NewStore returns a Store retaining at most max alerts (oldest dropped).
func NewStore(max int) *Store {
	if max <= 0 {
		max = 200
	}
	return &Store{
		max:      max,
		dedupe:   45 * time.Second,
		lastRule: make(map[string]time.Time),
	}
}

func dedupeKey(a Alert) string {
	if a.Ref != nil {
		if id := a.Ref["event_id"]; id != "" {
			return a.RuleID + "|evt|" + id
		}
	}
	return a.RuleID
}

// Add appends an alert, trimming history when over capacity.
func (s *Store) Add(a Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dedupe > 0 {
		k := dedupeKey(a)
		if prev, ok := s.lastRule[k]; ok && time.Since(prev) < s.dedupe {
			return
		}
		s.lastRule[k] = time.Now().UTC()
	}
	s.buf = append(s.buf, a)
	if len(s.buf) > s.max {
		s.buf = s.buf[len(s.buf)-s.max:]
	}
}

// Snapshot returns alerts newest-first (copy).
func (s *Store) Snapshot() []Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Alert, len(s.buf))
	copy(out, s.buf)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Len returns the number of buffered alerts.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.buf)
}

// Clear removes all alerts (operator action / demo reset).
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = nil
	s.lastRule = make(map[string]time.Time)
}
