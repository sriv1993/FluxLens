package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/httpauth"
	"github.com/sriharshav1/fluxlens/internal/stream"
)

func testRoutes(s *server) http.Handler {
	return httpauth.RBACMiddleware(nil, s.routes("api/openapi.yaml", stream.NewHub(), newLiveState(100), nil))
}

func TestVerticalSlice_IngestSuggestResolveExport(t *testing.T) {
	s := newServer(500, auditlog.NewChain(), defaultWedgeInstruction)
	ts := httptest.NewServer(withLogging(testRoutes(s)))
	defer ts.Close()
	c := ts.Client()

	e, err := canonical.NewEvent("line-east-1", canonical.SourceSynthetic, "sensor.variance", canonical.SeverityWarn, map[string]any{"lane": "B"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("events %s: %s", resp.Status, b)
	}
	var ack map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		t.Fatal(err)
	}
	eid := ack["event_id"]

	suggestBody, _ := json.Marshal(map[string]string{"event_id": eid})
	resp2, err := c.Post(ts.URL+"/api/v1/operator/suggest", "application/json", bytes.NewReader(suggestBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("suggest %s: %s", resp2.Status, b)
	}
	var sug map[string]json.RawMessage
	if err := json.NewDecoder(resp2.Body).Decode(&sug); err != nil {
		t.Fatal(err)
	}
	var dec struct {
		AuditChainHash string `json:"audit_chain_hash"`
	}
	if err := json.Unmarshal(sug["decision"], &dec); err != nil {
		t.Fatal(err)
	}
	if dec.AuditChainHash == "" {
		t.Fatal("missing audit_chain_hash")
	}

	resolveBody, _ := json.Marshal(map[string]string{
		"event_id":            eid,
		"decision_audit_hash": dec.AuditChainHash,
		"operator_id":         "jane-demo",
		"action":              "accept",
		"annotation":          "confirmed via automated wedge test",
	})
	resp3, err := c.Post(ts.URL+"/api/v1/operator/resolve", "application/json", bytes.NewReader(resolveBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp3.Body)
		t.Fatalf("resolve %s: %s", resp3.Status, b)
	}

	resp4, err := c.Get(ts.URL + "/api/v1/operator/export")
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("export %s", resp4.Status)
	}
	var bundle map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&bundle); err != nil {
		t.Fatal(err)
	}
	if bundle["records"] == nil {
		t.Fatal("bundle missing records")
	}
}

func TestOperatorSuggestPrecedents(t *testing.T) {
	s := newServer(500, auditlog.NewChain(), defaultWedgeInstruction)
	ts := httptest.NewServer(withLogging(testRoutes(s)))
	defer ts.Close()
	c := ts.Client()

	e, err := canonical.NewEvent("line-east-1", canonical.SourceSynthetic, "sensor.variance", canonical.SeverityCritical, map[string]any{"lane": "B"})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(e)
	resp, err := c.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var ack map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&ack)
	resp.Body.Close()
	eid := ack["event_id"]

	body, _ := json.Marshal(map[string]any{"event_id": eid, "max_precedents": 3})
	resp2, err := c.Post(ts.URL+"/api/v1/operator/suggest-precedents", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("suggest-precedents %s: %s", resp2.Status, b)
	}
	var sug struct {
		Steps []struct {
			Text               string `json:"text"`
			CitedPrecedentHash string `json:"cited_precedent_hash"`
		} `json:"steps"`
		Decision struct {
			AuditChainHash string `json:"audit_chain_hash"`
		} `json:"decision"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&sug); err != nil {
		t.Fatal(err)
	}
	if len(sug.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	if sug.Decision.AuditChainHash == "" {
		t.Fatal("missing decision audit hash")
	}
}
