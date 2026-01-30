// Package e2e contains in-process end-to-end tests of the FluxLens
// pipeline: synthetic events → curation → AI orchestration → audit
// log. These tests do not require Kafka, Postgres, or any external
// dependency. The full Kafka-based integration test lives separately
// and runs against the docker-compose stack.
package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/curation"
	"github.com/sriharshav1/fluxlens/internal/llm"
	"github.com/sriharshav1/fluxlens/internal/orchestrator"
)

func generateEvents(t *testing.T, sources, perSource int) []canonical.Event {
	t.Helper()
	out := make([]canonical.Event, 0, sources*perSource)
	age := 0
	for s := 1; s <= sources; s++ {
		for i := 0; i < perSource; i++ {
			e, err := canonical.NewEvent(
				"e2e-src-"+itoa(s),
				canonical.SourceSynthetic,
				"e2e.tick",
				canonical.SeverityInfo,
				map[string]int{"i": i},
			)
			if err != nil {
				t.Fatal(err)
			}
			e.Timestamp = time.Now().Add(-time.Duration(age) * time.Second)
			out = append(out, e)
			age++
		}
	}
	return out
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	a := i / 10
	b := i % 10
	return string(rune('0'+a)) + string(rune('0'+b))
}

func TestEndToEnd_HappyPath(t *testing.T) {
	events := generateEvents(t, 8, 6) // 48 events from 8 sources

	res, err := curation.Select(curation.Request{
		Strategy:         curation.StrategyGuaranteedMinDiversity,
		Events:           events,
		K:                10,
		DiversityPercent: 80,
		Now:              time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Selected) == 0 {
		t.Fatal("curator returned no events")
	}
	if res.DiversityScore < 0.5 {
		t.Fatalf("low diversity %f", res.DiversityScore)
	}

	chain := auditlog.NewChain()
	provider := llm.NewMockProvider("e2e-mock", llm.DecisionResponse{
		Classification: "routine",
		Suggestion:     "continue monitoring",
		Confidence:     0.88,
		RequiresReview: false,
	})
	orc := orchestrator.New(provider, orchestrator.NewDefaultGuardrails(), chain, "e2e")

	for _, e := range res.Selected {
		dec, err := orc.Decide(context.Background(), e, "Classify and suggest operator action.")
		if err != nil {
			t.Fatalf("decide: %v", err)
		}
		if dec.Guardrails != "pass" {
			t.Fatalf("expected pass got %q", dec.Guardrails)
		}
	}

	if chain.Len() != len(res.Selected) {
		t.Fatalf("expected %d audit records got %d", len(res.Selected), chain.Len())
	}
	ok, err := chain.Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("audit chain should verify after end-to-end run")
	}
}

func TestEndToEnd_OperatorOverrideIsAudited(t *testing.T) {
	chain := auditlog.NewChain()
	provider := llm.NewMockProvider("e2e-mock")
	orc := orchestrator.New(provider, orchestrator.NewDefaultGuardrails(), chain, "e2e")

	events := generateEvents(t, 1, 1)
	_, err := orc.Decide(context.Background(), events[0], "Classify.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orc.RecordOperatorAction("dec-001", "operator-jane", "override", "manually escalated"); err != nil {
		t.Fatal(err)
	}
	if chain.Len() != 2 {
		t.Fatalf("expected 2 records (decision + operator_action) got %d", chain.Len())
	}
	ok, _ := chain.Verify()
	if !ok {
		t.Fatal("chain should verify with operator_action present")
	}
}
