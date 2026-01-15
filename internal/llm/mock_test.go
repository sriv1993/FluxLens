package llm

import (
	"context"
	"testing"
)

func TestMockProviderDefault(t *testing.T) {
	m := NewMockProvider("test")
	r, err := m.Decide(context.Background(), DecisionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Classification == "" {
		t.Fatal("expected default classification")
	}
	if m.Calls() != 1 {
		t.Fatalf("expected 1 call got %d", m.Calls())
	}
}

func TestMockProviderCycles(t *testing.T) {
	m := NewMockProvider("test",
		DecisionResponse{Classification: "a", Confidence: 0.9},
		DecisionResponse{Classification: "b", Confidence: 0.7},
		DecisionResponse{Classification: "c", Confidence: 0.5, RequiresReview: true},
	)
	got := make([]string, 0)
	for i := 0; i < 6; i++ {
		r, _ := m.Decide(context.Background(), DecisionRequest{})
		got = append(got, r.Classification)
	}
	want := []string{"a", "b", "c", "a", "b", "c"}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestMockProviderFailNext(t *testing.T) {
	m := NewMockProvider("test")
	m.FailNextCall()
	if _, err := m.Decide(context.Background(), DecisionRequest{}); err == nil {
		t.Fatal("expected injected failure")
	}
	if _, err := m.Decide(context.Background(), DecisionRequest{}); err != nil {
		t.Fatalf("subsequent call should succeed: %v", err)
	}
}
