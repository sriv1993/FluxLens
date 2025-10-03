// Command api-gateway exposes the FluxLens REST API. Phase 1 implements
// the health endpoint, a digest request endpoint that proxies into the
// curation library, and an audit-snapshot endpoint backed by the
// in-memory chain. Authentication, RBAC, and WebSocket fan-out are
// added in Phase 2 (milestones M2.8 and onward).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/curation"
)

type server struct {
	mu        sync.RWMutex
	recent    []canonical.Event
	chain     *auditlog.Chain
	maxRecent int
}

func newServer(maxRecent int) *server {
	return &server{
		chain:     auditlog.NewChain(),
		maxRecent: maxRecent,
	}
}

func (s *server) addRecent(e canonical.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recent = append(s.recent, e)
	if len(s.recent) > s.maxRecent {
		s.recent = s.recent[len(s.recent)-s.maxRecent:]
	}
}

func (s *server) snapshotRecent() []canonical.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]canonical.Event, len(s.recent))
	copy(out, s.recent)
	return out
}

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	maxRecent := flag.Int("max-recent", 5000, "Max recent events to keep in memory for /digest")
	flag.Parse()

	srv := newServer(*maxRecent)

	// Seed the audit chain with a startup record so operators can see the chain working.
	_, _ = srv.chain.Append("system", map[string]string{
		"event": "api-gateway-startup",
		"time":  time.Now().UTC().Format(time.RFC3339Nano),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", srv.handleHealth)
	mux.HandleFunc("/api/v1/events", srv.handleEvents)
	mux.HandleFunc("/api/v1/digest", srv.handleDigest)
	mux.HandleFunc("/api/v1/audit", srv.handleAudit)

	h := &http.Server{
		Addr:              *addr,
		Handler:           withLogging(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	go func() {
		log.Printf("api-gateway listening on %s", *addr)
		if err := h.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = h.Shutdown(shutdownCtx)
	log.Println("api-gateway shut down")
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	verified, err := s.chain.Verify()
	resp := map[string]any{
		"status":             "ok",
		"audit_chain_length": s.chain.Len(),
		"audit_chain_head":   s.chain.HeadHash(),
		"audit_chain_valid":  verified,
		"audit_chain_error":  errString(err),
		"recent_events":      len(s.snapshotRecent()),
		"build_time":         time.Now().UTC().Format(time.RFC3339Nano),
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleEvents POST accepts a single canonical event and adds it to the
// in-memory recent-events buffer. Useful for local smoke testing without
// running the full ingest pipeline.
func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var e canonical.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := e.Validate(); err != nil {
		http.Error(w, "invalid event: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.addRecent(e)
	_, _ = s.chain.Append("ingest", map[string]string{
		"event_id":  e.EventID,
		"source_id": e.SourceID,
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"event_id": e.EventID})
}

func (s *server) handleDigest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	strategy, _ := strconv.Atoi(q.Get("strategy"))
	if strategy < 1 || strategy > 6 {
		strategy = int(curation.StrategyGuaranteedMinDiversity)
	}
	diversity, err := strconv.ParseFloat(q.Get("diversity"), 64)
	if err != nil {
		diversity = 80
	}
	k, err := strconv.Atoi(q.Get("k"))
	if err != nil || k <= 0 {
		k = 20
	}
	req := curation.Request{
		Strategy:         curation.Strategy(strategy),
		Events:           s.snapshotRecent(),
		K:                k,
		DiversityPercent: diversity,
		Now:              time.Now(),
	}
	res, err := curation.Select(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = s.chain.Append("decision", map[string]any{
		"strategy":   strategy,
		"diversity":  diversity,
		"k":          k,
		"selected_n": len(res.Selected),
	})
	writeJSON(w, http.StatusOK, res)
}

func (s *server) handleAudit(w http.ResponseWriter, _ *http.Request) {
	verified, err := s.chain.Verify()
	writeJSON(w, http.StatusOK, map[string]any{
		"verified":   verified,
		"verify_err": errString(err),
		"head_hash":  s.chain.HeadHash(),
		"records":    s.chain.Snapshot(),
	})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}
