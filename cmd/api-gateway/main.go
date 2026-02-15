// Command api-gateway exposes the FluxLens REST and WebSocket API,
// optional Kafka bridge (decisions + curated + raw topics), webhook
// ingest, API-key auth, and role-based access control.
//
// The gateway also wires one production-grade vertical slice for demos:
// ingest → digest → mock LLM suggestion → human accept/override → audit export.
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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sriharshav1/fluxlens/internal/alerts"
	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/auditlog/factory"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/curation"
	"github.com/sriharshav1/fluxlens/internal/httpauth"
	"github.com/sriharshav1/fluxlens/internal/llm"
	"github.com/sriharshav1/fluxlens/internal/observability"
	"github.com/sriharshav1/fluxlens/internal/kafkabridge"
	"github.com/sriharshav1/fluxlens/internal/orchestrator"
	"github.com/sriharshav1/fluxlens/internal/stream"
	"github.com/sriharshav1/fluxlens/pkg/domainpack"
)

// defaultWedgeInstruction is the bundled domain-pack prompt for the demo slice.
const defaultWedgeInstruction = `You are assisting a manufacturing line supervisor.
Classify the event (routine / elevated / critical) and propose one concrete operator action in under two sentences.
If evidence is ambiguous, bias toward requesting human review rather than autonomous operational moves.`

type server struct {
	mu               sync.RWMutex
	recent           []canonical.Event
	chain            auditlog.Store
	maxRecent        int
	alerts           *alerts.Store
	auditOK          bool
	orch             *orchestrator.Orchestrator
	wedgeInstruction string
	hub              *stream.Hub
}

func newServer(maxRecent int, chain auditlog.Store, wedgeInstruction string) *server {
	if chain == nil {
		chain = auditlog.NewChain()
	}
	mock := llm.NewMockProvider(
		"fluxlens-wedge-v1",
		llm.DecisionResponse{
			Classification: "routine",
			Suggestion:     "Continue monitoring; no line stop recommended for this anomaly signature.",
			Confidence:     0.84,
			RequiresReview: false,
			Reasons:        []string{"within historical variance"},
		},
		llm.DecisionResponse{
			Classification: "elevated",
			Suggestion:     "Pause outbound shipments from the affected SKU lane until QA clears lot sampling.",
			Confidence:     0.68,
			RequiresReview: true,
			Reasons:        []string{"confidence below autonomous-release threshold"},
		},
	)
	return &server{
		chain:            chain,
		maxRecent:        maxRecent,
		alerts:           alerts.NewStore(400),
		auditOK:          true,
		orch:             orchestrator.New(mock, nil, chain, "demo"),
		wedgeInstruction: wedgeInstruction,
	}
}

func (s *server) routes(openAPIPath string, hub *stream.Hub, live *liveState, allowedKeys map[string]struct{}) http.Handler {
	mux := http.NewServeMux()
	operator := httpauth.RequireRoles(httpauth.RoleOperator, httpauth.RoleReviewer, httpauth.RoleAdmin)
	auditor := httpauth.RequireRoles(httpauth.RoleAuditor, httpauth.RoleAdmin)

	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/events", s.handleEvents)
	mux.HandleFunc("/api/v1/digest", s.handleDigest)
	mux.HandleFunc("/api/v1/audit", s.handleAudit)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	mux.Handle("/api/v1/operator/suggest", operator(http.HandlerFunc(s.handleOperatorSuggest)))
	mux.Handle("/api/v1/operator/suggest-precedents", operator(http.HandlerFunc(s.handleOperatorSuggestPrecedents)))
	mux.Handle("/api/v1/operator/resolve", operator(http.HandlerFunc(s.handleOperatorResolve)))
	mux.Handle("/api/v1/operator/export", auditor(http.HandlerFunc(s.handleOperatorExport)))
	mux.HandleFunc("/api/v1/stream", func(w http.ResponseWriter, r *http.Request) {
		s.handleStream(w, r, hub, allowedKeys)
	})
	mux.HandleFunc("/api/v1/webhook", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebhook(w, r, hub)
	})
	mux.HandleFunc("/api/v1/decisions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleDecisions(w, r, live)
	})
	mux.Handle("/metrics", observability.Handler())
	mux.HandleFunc("/api/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, openAPIPath)
	})
	return mux
}

func (s *server) observeAudit(ok bool, verifyErr error) {
	valid := ok && verifyErr == nil
	if !valid && s.auditOK {
		for _, a := range alerts.FromAuditBroken(verifyErr) {
			s.alerts.Add(a)
		}
	}
	s.auditOK = valid
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

func (s *server) findEvent(id string) (canonical.Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.recent {
		if e.EventID == id {
			return e, true
		}
	}
	return canonical.Event{}, false
}

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	maxRecent := flag.Int("max-recent", 5000, "Max recent events to keep in memory for /digest")
	postgresDSN := flag.String("postgres-dsn", os.Getenv("FLUXLENS_POSTGRES_DSN"), "Postgres DSN for durable audit log; empty uses in-memory")
	tenant := flag.String("tenant", "default", "Tenant id when using Postgres audit log")
	apiKeysFlag := flag.String("api-keys", "", "Comma-separated API keys (or set FLUXLENS_API_KEYS)")
	openAPIPath := flag.String("openapi", "api/openapi.yaml", "Path to OpenAPI spec file")
	domainPackPath := flag.String("domain-pack", "", "YAML domain pack for default LLM instruction")
	apiKeyRolesFlag := flag.String("api-key-roles", "", "key:role+role,... (or FLUXLENS_API_KEY_ROLES)")
	kafkaAddr := flag.String("kafka", os.Getenv("FLUXLENS_KAFKA"), "Kafka bootstrap; enables decisions/curated/raw bridge when set")
	kafkaGroup := flag.String("kafka-group", "fluxlens-gateway", "Consumer group for gateway Kafka bridge")
	decisionsTopic := flag.String("kafka-decisions-topic", "fluxlens.decisions", "Kafka topic for orchestrator decisions")
	curatedTopic := flag.String("kafka-curated-topic", "fluxlens.events.curated", "Kafka topic for curated digests")
	rawTopic := flag.String("kafka-raw-topic", "fluxlens.events.raw", "Kafka topic for raw canonical events")
	flag.Parse()

	openCtx, openCancel := context.WithTimeout(context.Background(), 15*time.Second)
	chain, closeStore, err := factory.Open(openCtx, *postgresDSN, *tenant)
	openCancel()
	if err != nil {
		log.Fatalf("audit store: %v", err)
	}
	defer closeStore()

	wedgeInstr := defaultWedgeInstruction
	if *domainPackPath != "" {
		pack, err := domainpack.Load(*domainPackPath)
		if err != nil {
			log.Fatalf("domain pack: %v", err)
		}
		wedgeInstr = pack.DefaultInstruction(defaultWedgeInstruction)
		log.Printf("loaded domain pack %q", pack.Name)
	}

	hub := stream.NewHub()
	srv := newServer(*maxRecent, chain, wedgeInstr)
	srv.hub = hub
	live := newLiveState(500)

	// Seed the audit chain with a startup record so operators can see the chain working.
	_, _ = srv.chain.Append("system", map[string]string{
		"event":   "api-gateway-startup",
		"time":    time.Now().UTC().Format(time.RFC3339Nano),
		"backend": factory.BackendLabel(chain),
	})

	apiKeys := httpauth.SplitKeys(*apiKeysFlag)
	if len(apiKeys) == 0 {
		apiKeys = httpauth.KeysFromEnv()
	}
	keyRoles := httpauth.ParseKeyRoles(httpauth.SplitKeys(*apiKeyRolesFlag))
	if len(keyRoles) == 0 {
		keyRoles = httpauth.ParseKeyRoles(httpauth.RolesFromEnv())
	}
	for _, k := range apiKeys {
		if _, ok := keyRoles[k]; !ok {
			keyRoles[k] = []string{httpauth.RoleOperator}
		}
	}
	if len(apiKeys) == 0 && len(keyRoles) > 0 {
		for k := range keyRoles {
			apiKeys = append(apiKeys, k)
		}
	}
	allowedKeys := httpauth.KeysToSet(apiKeys)
	handler := observability.Instrument(
		httpauth.APIKeyMiddleware(apiKeys,
			httpauth.RBACMiddleware(keyRoles,
				srv.routes(*openAPIPath, hub, live, allowedKeys),
			),
		),
	)
	if len(apiKeys) > 0 {
		log.Printf("API key authentication enabled (%d keys)", len(apiKeys))
	}
	if len(keyRoles) > 0 {
		log.Printf("RBAC enabled for %d principals", len(keyRoles))
	}

	h := &http.Server{
		Addr:              *addr,
		Handler:           withLogging(handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	if strings.TrimSpace(*kafkaAddr) != "" {
		srv.startKafkaBridge(ctx, kafkabridge.Config{
			Brokers:        *kafkaAddr,
			GroupID:        *kafkaGroup,
			DecisionsTopic: *decisionsTopic,
			CuratedTopic:   *curatedTopic,
			RawTopic:       *rawTopic,
		}, hub, live)
		log.Printf("Kafka bridge enabled (brokers=%s decisions=%s)", *kafkaAddr, *decisionsTopic)
	}

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
	s.observeAudit(verified, err)
	observability.SetAuditMetrics(s.chain.Len(), verified && err == nil)
	resp := map[string]any{
		"status":             "ok",
		"audit_backend":      factory.BackendLabel(s.chain),
		"audit_chain_length": s.chain.Len(),
		"audit_chain_head":   s.chain.HeadHash(),
		"audit_chain_valid":  verified,
		"audit_chain_error":  errString(err),
		"recent_events":      len(s.snapshotRecent()),
		"alerts_buffered":    s.alerts.Len(),
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
	s.ingestEvent(e, s.hub)
	ok, verr := s.chain.Verify()
	s.observeAudit(ok, verr)
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
	_, _ = s.chain.Append("digest_selection", map[string]any{
		"strategy":   strategy,
		"diversity":  diversity,
		"k":          k,
		"selected_n": len(res.Selected),
	})
	for _, a := range alerts.FromDigestQuality(res.FreshnessScore, res.DiversityScore, strategy, k, len(res.Selected)) {
		s.alerts.Add(a)
	}
	ok, verr := s.chain.Verify()
	s.observeAudit(ok, verr)
	if s.hub != nil {
		s.hub.BroadcastJSON(stream.TypeDigest, res)
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *server) handleOperatorSuggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		EventID     string `json:"event_id"`
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.EventID == "" {
		http.Error(w, "event_id required", http.StatusBadRequest)
		return
	}
	ev, ok := s.findEvent(req.EventID)
	if !ok {
		http.Error(w, "unknown event_id — POST /api/v1/events first", http.StatusNotFound)
		return
	}
	instr := strings.TrimSpace(req.Instruction)
	if instr == "" {
		instr = s.wedgeInstruction
	}
	dec, err := s.orch.Decide(r.Context(), ev, instr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dec.OperatorReview {
		class := dec.Response.Classification
		if strings.TrimSpace(class) == "" {
			class = dec.Guardrails
		}
		for _, a := range alerts.FromOperatorReviewRequired(dec.EventID, class) {
			s.alerts.Add(a)
		}
	}
	vok, verr := s.chain.Verify()
	s.observeAudit(vok, verr)
	writeJSON(w, http.StatusOK, map[string]any{"decision": dec})
}

func (s *server) handleOperatorSuggestPrecedents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		EventID       string `json:"event_id"`
		Instruction   string `json:"instruction"`
		MaxPrecedents int    `json:"max_precedents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.EventID == "" {
		http.Error(w, "event_id required", http.StatusBadRequest)
		return
	}
	ev, ok := s.findEvent(req.EventID)
	if !ok {
		http.Error(w, "unknown event_id — POST /api/v1/events first", http.StatusNotFound)
		return
	}
	instr := strings.TrimSpace(req.Instruction)
	if instr == "" {
		instr = orchestrator.DefaultPrecedentInstruction
	}
	sug, err := s.orch.SuggestWithPrecedents(r.Context(), ev, instr, req.MaxPrecedents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sug.Decision.OperatorReview {
		class := sug.Decision.Response.Classification
		if strings.TrimSpace(class) == "" {
			class = sug.Decision.Guardrails
		}
		for _, a := range alerts.FromOperatorReviewRequired(sug.Decision.EventID, class) {
			s.alerts.Add(a)
		}
	}
	vok, verr := s.chain.Verify()
	s.observeAudit(vok, verr)
	writeJSON(w, http.StatusOK, sug)
}

type resolveBody struct {
	EventID           string `json:"event_id"`
	DecisionAuditHash string `json:"decision_audit_hash"`
	OperatorID        string `json:"operator_id"`
	Action            string `json:"action"`
	Annotation        string `json:"annotation"`
}

func (s *server) handleOperatorResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req resolveBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.EventID == "" || req.DecisionAuditHash == "" || req.OperatorID == "" {
		http.Error(w, "event_id, decision_audit_hash, and operator_id are required", http.StatusBadRequest)
		return
	}
	if req.Action != "accept" && req.Action != "override" && req.Action != "annotate" {
		http.Error(w, "action must be accept, override, or annotate", http.StatusBadRequest)
		return
	}
	h, err := s.orch.RecordOperatorAction(req.DecisionAuditHash, req.OperatorID, req.Action, req.Annotation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, a := range alerts.FromOperatorResolution(req.EventID, req.OperatorID, req.Action) {
		s.alerts.Add(a)
	}
	vok, verr := s.chain.Verify()
	s.observeAudit(vok, verr)
	writeJSON(w, http.StatusOK, map[string]any{"operator_audit_hash": h})
}

func (s *server) handleOperatorExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	verified, err := s.chain.Verify()
	s.observeAudit(verified, err)
	writeJSON(w, http.StatusOK, map[string]any{
		"slice":           "ingest_digest_ai_operator_export",
		"exported_at":     time.Now().UTC().Format(time.RFC3339Nano),
		"audit_verified":  verified && err == nil,
		"verify_err":      errString(err),
		"chain_head_hash": s.chain.HeadHash(),
		"records":         s.chain.Snapshot(),
	})
}

func (s *server) handleAudit(w http.ResponseWriter, _ *http.Request) {
	verified, err := s.chain.Verify()
	s.observeAudit(verified, err)
	writeJSON(w, http.StatusOK, map[string]any{
		"verified":   verified,
		"verify_err": errString(err),
		"head_hash":  s.chain.HeadHash(),
		"records":    s.chain.Snapshot(),
	})
}

func (s *server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"alerts": s.alerts.Snapshot(),
			"count":  s.alerts.Len(),
		})
	case http.MethodDelete:
		s.alerts.Clear()
		writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
