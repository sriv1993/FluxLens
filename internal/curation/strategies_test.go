package curation

import (
	"math/rand"
	"testing"
	"time"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

// helper: build an event with a given source and timestamp offset (sec ago).
func ev(src string, secAgo int) canonical.Event {
	e, _ := canonical.NewEvent(src, canonical.SourceSynthetic, "tick", canonical.SeverityInfo, nil)
	e.Timestamp = time.Now().Add(-time.Duration(secAgo) * time.Second)
	return e
}

func evSet() []canonical.Event {
	// 5 sources, 4 events each, varying ages (1..20s ago)
	out := make([]canonical.Event, 0, 20)
	age := 1
	for s := 1; s <= 5; s++ {
		for i := 0; i < 4; i++ {
			out = append(out, ev("s"+string(rune('0'+s)), age))
			age++
		}
	}
	return out
}

func TestSelectLatest(t *testing.T) {
	events := evSet()
	r, err := Select(Request{Strategy: StrategyLatest, Events: events, K: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Selected) != 5 {
		t.Fatalf("expected 5 got %d", len(r.Selected))
	}
	// First selected should be the freshest event overall.
	if r.Selected[0].Timestamp != mostRecent(events).Timestamp {
		t.Fatal("first selected should be most recent")
	}
}

func TestSelectLatestPerSource(t *testing.T) {
	events := evSet()
	r, err := Select(Request{Strategy: StrategyLatestPerSource, Events: events, K: 10})
	if err != nil {
		t.Fatal(err)
	}
	// Must contain at most one event per source.
	seen := make(map[string]int)
	for _, e := range r.Selected {
		seen[e.SourceID]++
	}
	for src, n := range seen {
		if n > 1 {
			t.Fatalf("source %s appears %d times", src, n)
		}
	}
	// Diversity score for 5/5 sources should be 1.0.
	if r.DiversityScore < 0.99 {
		t.Fatalf("expected diversity ~1.0 got %f", r.DiversityScore)
	}
}

func TestSelectHybrid(t *testing.T) {
	events := evSet()
	r, err := Select(Request{Strategy: StrategyHybridLatestAndPerSource, Events: events, K: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Selected) != 8 {
		t.Fatalf("expected 8 got %d", len(r.Selected))
	}
	// All sources should be represented (5 unique).
	srcs := uniqueSources(r.Selected)
	if srcs < 5 {
		t.Fatalf("expected all 5 sources represented got %d", srcs)
	}
}

func TestGuaranteedDiversity_HighFloor(t *testing.T) {
	events := evSet()
	r, err := Select(Request{
		Strategy:         StrategyGuaranteedMinDiversity,
		Events:           events,
		K:                10,
		DiversityPercent: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if uniqueSources(r.Selected) < 5 {
		t.Fatal("100% diversity should include all sources")
	}
}

func TestGuaranteedDiversity_LowFloor(t *testing.T) {
	events := evSet()
	r, err := Select(Request{
		Strategy:         StrategyGuaranteedMinDiversity,
		Events:           events,
		K:                10,
		DiversityPercent: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	// With low diversity floor, freshness should be high (close to picking globally latest).
	if r.FreshnessScore < 0.5 {
		t.Fatalf("expected high freshness got %f", r.FreshnessScore)
	}
}

func TestGuaranteedDiversityRandom_Determinism(t *testing.T) {
	events := evSet()
	rng := rand.New(rand.NewSource(42)) //nolint:gosec
	r1, _ := Select(Request{
		Strategy:         StrategyGuaranteedMinDiversityRandom,
		Events:           events,
		K:                5,
		DiversityPercent: 60,
		Rand:             rng,
	})
	rng = rand.New(rand.NewSource(42)) //nolint:gosec
	r2, _ := Select(Request{
		Strategy:         StrategyGuaranteedMinDiversityRandom,
		Events:           events,
		K:                5,
		DiversityPercent: 60,
		Rand:             rng,
	})
	if len(r1.Selected) != len(r2.Selected) {
		t.Fatal("seeded RNG must yield deterministic results")
	}
}

func TestPreferredSources_InjectsWhenAbsent(t *testing.T) {
	events := evSet()
	// Pick a source that would normally be evicted at low diversity.
	r, err := Select(Request{
		Strategy:         StrategyPreferredSources,
		Events:           events,
		K:                3,
		DiversityPercent: 20,
		PreferredSources: []string{"s5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range r.Selected {
		if e.SourceID == "s5" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("preferred source s5 should be present in result")
	}
}

func TestRedundancySuppression(t *testing.T) {
	events := evSet()
	suppress := map[string]struct{}{
		events[0].EventID: {},
		events[1].EventID: {},
	}
	r, err := Select(Request{
		Strategy:       StrategyLatest,
		Events:         events,
		K:              5,
		SuppressionSet: suppress,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range r.Selected {
		if _, hit := suppress[e.EventID]; hit {
			t.Fatal("suppressed event leaked into result")
		}
	}
	if r.RedundancyScore <= 0 {
		t.Fatal("redundancy score should reflect filtered events")
	}
}

func TestUnknownStrategy(t *testing.T) {
	_, err := Select(Request{Strategy: Strategy(99), Events: evSet(), K: 3})
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func mostRecent(events []canonical.Event) canonical.Event {
	out := events[0]
	for _, e := range events[1:] {
		if e.Timestamp.After(out.Timestamp) {
			out = e
		}
	}
	return out
}

func uniqueSources(events []canonical.Event) int {
	seen := make(map[string]struct{})
	for _, e := range events {
		seen[e.SourceID] = struct{}{}
	}
	return len(seen)
}
