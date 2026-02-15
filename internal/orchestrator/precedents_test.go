package orchestrator

import (
	"context"
	"testing"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/llm"
)

func TestSuggestWithPrecedents_IncludesPrecedentsInSteps(t *testing.T) {
	chain := auditlog.NewChain()
	rec, _ := chain.Append("decision", map[string]any{
		"event_id":          "old",
		"event_type":        "sensor.variance",
		"source_id":         "line-a",
		"severity":          "critical",
		"tenant":            "demo",
		"guardrails_status": "pass",
		"response": map[string]any{
			"classification": "elevated",
			"suggestion":     "hold lane",
			"confidence":     0.9,
		},
	})
	_, _ = chain.Append("operator_action", map[string]any{
		"decision_id": rec.Hash,
		"operator_id": "op1",
		"action":      "override",
		"annotation":  "manual QA sign-off",
	})

	provider := llm.NewMockProvider("mock-p", llm.DecisionResponse{
		Classification: "critical",
		Suggestion:     "stop line until inspection",
		Confidence:     0.88,
		RequiresReview: true,
	})
	orc := New(provider, NewDefaultGuardrails(), chain, "demo")

	ev, err := canonical.NewEvent("line-a", canonical.SourceSynthetic, "sensor.variance", canonical.SeverityCritical, map[string]any{"lane": "B"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := orc.SuggestWithPrecedents(context.Background(), ev, "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Steps) < 2 {
		t.Fatalf("expected primary + precedent steps, got %d", len(out.Steps))
	}
	if out.Steps[1].CitedPrecedentHash == "" {
		t.Fatal("precedent step should cite decision hash")
	}
	if out.Decision.AuditChainHash == "" {
		t.Fatal("missing audit hash")
	}
	found := false
	for _, r := range chain.Snapshot() {
		if r.Kind == "decision_with_precedents" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected decision_with_precedents audit record")
	}
}
