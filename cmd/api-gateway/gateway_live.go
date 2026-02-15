package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sriharshav1/fluxlens/internal/alerts"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/curation"
	"github.com/sriharshav1/fluxlens/internal/httpauth"
	"github.com/sriharshav1/fluxlens/internal/kafkabridge"
	"github.com/sriharshav1/fluxlens/internal/orchestrator"
	"github.com/sriharshav1/fluxlens/internal/stream"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type liveState struct {
	mu          sync.RWMutex
	decisions   []orchestrator.Decision
	lastDigest  *curation.Result
	maxDecisions int
}

func newLiveState(max int) *liveState {
	if max <= 0 {
		max = 500
	}
	return &liveState{maxDecisions: max}
}

func (ls *liveState) addDecision(d orchestrator.Decision) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.decisions = append(ls.decisions, d)
	if len(ls.decisions) > ls.maxDecisions {
		ls.decisions = ls.decisions[len(ls.decisions)-ls.maxDecisions:]
	}
}

func (ls *liveState) setDigest(res curation.Result) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	cp := res
	ls.lastDigest = &cp
}

func (ls *liveState) snapshotDecisions() []orchestrator.Decision {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	out := make([]orchestrator.Decision, len(ls.decisions))
	copy(out, ls.decisions)
	return out
}

func (ls *liveState) snapshotDigest() *curation.Result {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	if ls.lastDigest == nil {
		return nil
	}
	cp := *ls.lastDigest
	return &cp
}

func (s *server) ingestEvent(e canonical.Event, hub *stream.Hub) {
	s.addRecent(e)
	if hub != nil {
		hub.BroadcastJSON(stream.TypeEvent, e)
	}
	_, _ = s.chain.Append("ingest", map[string]string{
		"event_id":  e.EventID,
		"source_id": e.SourceID,
	})
	for _, a := range alerts.FromIngestEvent(e) {
		s.alerts.Add(a)
	}
}

func (s *server) startKafkaBridge(ctx context.Context, cfg kafkabridge.Config, hub *stream.Hub, live *liveState) {
	if strings.TrimSpace(cfg.Brokers) == "" {
		return
	}
	h := kafkabridge.Handlers{
		OnDecision: func(d orchestrator.Decision) {
			live.addDecision(d)
			if hub != nil {
				hub.BroadcastJSON(stream.TypeDecision, d)
			}
			if ev, ok := s.findEvent(d.EventID); ok {
				s.ingestEvent(ev, nil)
			}
		},
		OnCurated: func(res curation.Result) {
			live.setDigest(res)
			for _, ev := range res.Selected {
				s.addRecent(ev)
			}
			if hub != nil {
				hub.BroadcastJSON(stream.TypeDigest, res)
			}
		},
		OnRawEvent: func(ev canonical.Event) {
			s.ingestEvent(ev, hub)
		},
	}
	go kafkabridge.RunDecisionsConsumer(ctx, cfg, h)
	go kafkabridge.RunCuratedConsumer(ctx, cfg, h)
	go kafkabridge.RunRawConsumer(ctx, cfg, h)
}

func (s *server) handleStream(w http.ResponseWriter, r *http.Request, hub *stream.Hub, allowedKeys map[string]struct{}) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(allowedKeys) > 0 {
		key := httpauth.ExtractKey(r)
		if _, ok := allowedKeys[key]; !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	hub.Register(conn)
	defer hub.Unregister(conn)
	_ = conn.SetReadDeadline(time.Time{})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *server) handleDecisions(w http.ResponseWriter, _ *http.Request, live *liveState) {
	writeJSON(w, http.StatusOK, map[string]any{
		"decisions": live.snapshotDecisions(),
		"count":     len(live.snapshotDecisions()),
	})
}

func (s *server) handleWebhook(w http.ResponseWriter, r *http.Request, hub *stream.Hub) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SourceID  string          `json:"source_id"`
		EventType string          `json:"event_type"`
		Severity  string          `json:"severity"`
		Payload   json.RawMessage `json:"payload"`
		Timestamp *time.Time      `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.SourceID == "" || body.EventType == "" {
		http.Error(w, "source_id and event_type required", http.StatusBadRequest)
		return
	}
	sev := canonical.Severity(body.Severity)
	if sev == "" {
		sev = canonical.SeverityInfo
	}
	var payload any = body.Payload
	if len(body.Payload) == 0 {
		payload = map[string]string{}
	}
	ev, err := canonical.NewEvent(body.SourceID, canonical.SourceWebhook, body.EventType, sev, payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Timestamp != nil {
		ev.Timestamp = body.Timestamp.UTC()
	}
	if err := ev.Validate(); err != nil {
		http.Error(w, "invalid event: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.ingestEvent(ev, hub)
	ok, verr := s.chain.Verify()
	s.observeAudit(ok, verr)
	writeJSON(w, http.StatusAccepted, map[string]string{"event_id": ev.EventID})
}
