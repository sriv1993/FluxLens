package precedents

import (
	"testing"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
)

func seedResolvedChain(t *testing.T, chain *auditlog.Chain, eventType, sourceID, severity, tenant string) string {
	t.Helper()
	resp := map[string]any{
		"classification":  "elevated",
		"suggestion":      "pause lane",
		"confidence":      0.9,
		"requires_review": false,
	}
	rec, err := chain.Append("decision", map[string]any{
		"event_id":          "ev-old",
		"event_type":        eventType,
		"source_id":         sourceID,
		"severity":          severity,
		"tenant":            tenant,
		"guardrails_status": "pass",
		"response":          resp,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = chain.Append("operator_action", map[string]any{
		"decision_id": rec.Hash,
		"operator_id": "op-a",
		"action":      "accept",
		"annotation":  "confirmed sampling hold",
		"tenant":      tenant,
	})
	if err != nil {
		t.Fatal(err)
	}
	return rec.Hash
}

func TestFindMatches_AllDimensions(t *testing.T) {
	chain := auditlog.NewChain()
	seedResolvedChain(t, chain, "sensor.variance", "line-east-1", "critical", "demo")
	seedResolvedChain(t, chain, "sensor.variance", "line-east-1", "warn", "demo")
	seedResolvedChain(t, chain, "sensor.variance", "line-west-2", "critical", "demo")

	got := FindMatches(chain, Criteria{
		EventType: "sensor.variance",
		SourceID:  "line-east-1",
		Severity:  "critical",
		TenantID:  "demo",
	}, 10)
	if len(got) != 1 {
		t.Fatalf("want 1 match got %d", len(got))
	}
	if got[0].OperatorAction != "accept" {
		t.Fatalf("unexpected action %q", got[0].OperatorAction)
	}
}

func TestFindMatches_RespectsMax(t *testing.T) {
	chain := auditlog.NewChain()
	for i := 0; i < 4; i++ {
		seedResolvedChain(t, chain, "tick", "src-1", "error", "default")
	}
	got := FindMatches(chain, Criteria{EventType: "tick", SourceID: "src-1", Severity: "error"}, 2)
	if len(got) != 2 {
		t.Fatalf("want 2 got %d", len(got))
	}
}

func TestFindMatches_NoOperatorActionSkipped(t *testing.T) {
	chain := auditlog.NewChain()
	_, _ = chain.Append("decision", map[string]any{
		"event_type":        "tick",
		"source_id":         "src-1",
		"severity":          "error",
		"tenant":            "default",
		"guardrails_status": "pass",
		"response":          map[string]any{"classification": "routine", "suggestion": "watch"},
	})
	got := FindMatches(chain, Criteria{EventType: "tick", SourceID: "src-1", Severity: "error"}, 5)
	if len(got) != 0 {
		t.Fatalf("expected no matches without operator_action, got %d", len(got))
	}
}
