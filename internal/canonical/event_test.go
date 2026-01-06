package canonical

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEventDefaults(t *testing.T) {
	e, err := NewEvent("src1", SourceMySQLCDC, "row_insert", SeverityInfo, map[string]int{"x": 1})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if e.EventID == "" {
		t.Fatal("expected non-empty EventID")
	}
	if e.SourceType != SourceMySQLCDC {
		t.Fatalf("unexpected SourceType %q", e.SourceType)
	}
	if e.Metadata.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %q got %q", SchemaVersion, e.Metadata.SchemaVersion)
	}
	if e.IngestedAt.IsZero() {
		t.Fatal("IngestedAt should be set")
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestNewEventRequiresSource(t *testing.T) {
	if _, err := NewEvent("", SourceWebhook, "x", SeverityInfo, nil); err == nil {
		t.Fatal("expected error for empty source_id")
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Event)
		wantErr bool
	}{
		{"valid", func(e *Event) {}, false},
		{"missing event_id", func(e *Event) { e.EventID = "" }, true},
		{"missing source_id", func(e *Event) { e.SourceID = "" }, true},
		{"missing source_type", func(e *Event) { e.SourceType = "" }, true},
		{"missing event_type", func(e *Event) { e.EventType = "" }, true},
		{"missing severity", func(e *Event) { e.Severity = "" }, true},
		{"unknown severity", func(e *Event) { e.Severity = "bogus" }, true},
		{"zero timestamp", func(e *Event) { e.Timestamp = time.Time{} }, true},
		{"missing schema version", func(e *Event) { e.Metadata.SchemaVersion = "" }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := NewEvent("src1", SourceMySQLCDC, "et", SeverityInfo, map[string]int{"x": 1})
			if err != nil {
				t.Fatalf("setup: %v", err)
			}
			tc.mutate(&e)
			err = e.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	e, err := NewEvent("src1", SourceMySQLCDC, "row_insert", SeverityWarn, map[string]int{"x": 7})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var e2 Event
	if err := json.Unmarshal(b, &e2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e2.EventID != e.EventID || e2.SourceID != e.SourceID || e2.Severity != e.Severity {
		t.Fatal("round-trip mismatch")
	}
}

func TestSeverityRanking(t *testing.T) {
	if SeverityCritical.Rank() <= SeverityInfo.Rank() {
		t.Fatal("critical should rank above info")
	}
	if Severity("nonsense").Rank() != -1 {
		t.Fatal("unknown severity should rank -1")
	}
}

func TestAgeAt(t *testing.T) {
	e, _ := NewEvent("s", SourceWebhook, "x", SeverityInfo, nil)
	e.Timestamp = time.Now().Add(-30 * time.Second)
	if e.AgeAt(time.Now()) < 25*time.Second {
		t.Fatal("expected age >= 25s")
	}
	// future event clamps to 0
	e.Timestamp = time.Now().Add(1 * time.Hour)
	if e.AgeAt(time.Now()) != 0 {
		t.Fatal("future event should age to 0")
	}
}
