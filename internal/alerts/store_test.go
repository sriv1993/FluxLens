package alerts

import (
	"testing"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

func TestStoreTrimAndOrder(t *testing.T) {
	s := NewStore(3)
	for i := 1; i <= 5; i++ {
		s.Add(Alert{
			ID:     string(rune('0' + i)),
			Title:  "t",
			RuleID: string(rune('a' + i)),
		})
	}
	if s.Len() != 3 {
		t.Fatalf("len=%d want 3", s.Len())
	}
	snap := s.Snapshot()
	if snap[0].ID != "5" || snap[2].ID != "3" {
		t.Fatalf("unexpected order %+v", snap)
	}
	s.Clear()
	if s.Len() != 0 {
		t.Fatal("clear failed")
	}
}

func TestFromIngestEventIgnoresInfo(t *testing.T) {
	e := canonical.Event{
		EventID:    "x",
		SourceID:   "s",
		SourceType: canonical.SourceSynthetic,
		EventType:  "heartbeat",
		Severity:   canonical.SeverityInfo,
		Metadata:   canonical.Metadata{SchemaVersion: canonical.SchemaVersion},
	}
	if len(FromIngestEvent(e)) != 0 {
		t.Fatal("expected no alerts")
	}
}

func TestFromIngestEventWarn(t *testing.T) {
	e := canonical.Event{
		EventID:    "evt-1",
		SourceID:   "src-a",
		SourceType: canonical.SourceSynthetic,
		EventType:  "alarm",
		Severity:   canonical.SeverityWarn,
		Metadata:   canonical.Metadata{SchemaVersion: canonical.SchemaVersion},
	}
	got := FromIngestEvent(e)
	if len(got) != 1 {
		t.Fatalf("want 1 alert got %d", len(got))
	}
	if got[0].RuleID != "ingest.severity_warn_or_above" {
		t.Fatal(got[0].RuleID)
	}
}

func TestFromDigestQualityFreshness(t *testing.T) {
	got := FromDigestQuality(0.1, 0.99, 4, 20, 18)
	var sawFresh bool
	for _, a := range got {
		if a.RuleID == "digest.low_freshness" {
			sawFresh = true
		}
	}
	if !sawFresh {
		t.Fatalf("missing freshness alert: %+v", got)
	}
}

func TestDedupeDigestRule(t *testing.T) {
	s := NewStore(50)
	a := FromDigestQuality(0.05, 0.99, 4, 20, 18)
	if len(a) == 0 {
		t.Fatal("expected freshness alert")
	}
	s.Add(a[0])
	s.Add(a[0])
	if s.Len() != 1 {
		t.Fatalf("dedupe failed len=%d", s.Len())
	}
}
