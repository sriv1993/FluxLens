// Package canonical defines the FluxLens canonical event schema and helpers.
//
// All events flowing through FluxLens — regardless of source type — are
// normalized to the Event type defined in this package before being published
// to the event bus. Downstream services (curator, AI orchestrator, audit
// writer) operate exclusively on canonical events.
package canonical

import (
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// SchemaVersion is the current canonical-event schema version. It follows
// semantic versioning. Schema changes must be backward compatible within a
// major version.
const SchemaVersion = "1.0.0"

// SourceType enumerates the recognized canonical source types. Operators
// can register additional source types via the plugin SDK; those types must
// be registered before events using them are validated.
type SourceType string

const (
	SourceMySQLCDC    SourceType = "mysql_cdc"
	SourcePostgresCDC SourceType = "postgres_cdc"
	SourceKafka       SourceType = "kafka"
	SourceWebhook     SourceType = "webhook"
	SourceSynthetic   SourceType = "synthetic"
)

// Severity enumerates canonical severity levels. Operators interpret these
// in the context of their domain pack; FluxLens does not impose semantics
// on severity beyond ordering.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Rank returns a numeric ordering for severities. Higher = more severe.
func (s Severity) Rank() int {
	switch s {
	case SeverityInfo:
		return 0
	case SeverityWarn:
		return 1
	case SeverityError:
		return 2
	case SeverityCritical:
		return 3
	default:
		return -1
	}
}

// Event is the canonical FluxLens event.
type Event struct {
	EventID    string          `json:"event_id"`
	SourceID   string          `json:"source_id"`
	SourceType SourceType      `json:"source_type"`
	EventType  string          `json:"event_type"`
	Severity   Severity        `json:"severity"`
	Timestamp  time.Time       `json:"timestamp"`
	IngestedAt time.Time       `json:"ingested_at"`
	Payload    json.RawMessage `json:"payload"`
	Metadata   Metadata        `json:"metadata"`
}

// Metadata carries platform-level context that is independent of the
// domain-specific Payload.
type Metadata struct {
	TraceID       string `json:"trace_id,omitempty"`
	IngestionPod  string `json:"ingestion_pod,omitempty"`
	SchemaVersion string `json:"schema_version"`
	TenantID      string `json:"tenant_id,omitempty"`
}

// NewEvent constructs a fresh canonical Event with a generated ULID,
// ingested timestamp set to now (UTC), and schema version set to the
// current SchemaVersion constant.
//
// The Timestamp field is set to now by default; if the source event has a
// distinct origin time, the caller should overwrite Timestamp before
// publishing.
func NewEvent(sourceID string, sourceType SourceType, eventType string, severity Severity, payload any) (Event, error) {
	if sourceID == "" {
		return Event{}, errors.New("canonical: source_id is required")
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("canonical: marshal payload: %w", err)
	}
	id, err := ulid.New(ulid.Now(), crand.Reader)
	if err != nil {
		return Event{}, fmt.Errorf("canonical: generate id: %w", err)
	}
	now := time.Now().UTC()
	return Event{
		EventID:    id.String(),
		SourceID:   sourceID,
		SourceType: sourceType,
		EventType:  eventType,
		Severity:   severity,
		Timestamp:  now,
		IngestedAt: now,
		Payload:    pb,
		Metadata:   Metadata{SchemaVersion: SchemaVersion},
	}, nil
}

// Validate returns a non-nil error if any required field is missing or
// malformed. It does not validate the domain-specific Payload structure;
// domain validation is the responsibility of registered domain packs.
func (e *Event) Validate() error {
	if e.EventID == "" {
		return errors.New("canonical: event_id required")
	}
	if e.SourceID == "" {
		return errors.New("canonical: source_id required")
	}
	if e.SourceType == "" {
		return errors.New("canonical: source_type required")
	}
	if e.EventType == "" {
		return errors.New("canonical: event_type required")
	}
	if e.Severity == "" {
		return errors.New("canonical: severity required")
	}
	if e.Severity.Rank() < 0 {
		return fmt.Errorf("canonical: unknown severity %q", e.Severity)
	}
	if e.Timestamp.IsZero() {
		return errors.New("canonical: timestamp required")
	}
	if e.Metadata.SchemaVersion == "" {
		return errors.New("canonical: metadata.schema_version required")
	}
	return nil
}

// AgeAt returns the time elapsed between the event's Timestamp and the
// given reference time. Negative values are returned as 0 (future events
// are treated as fresh).
func (e *Event) AgeAt(ref time.Time) time.Duration {
	d := ref.Sub(e.Timestamp)
	if d < 0 {
		return 0
	}
	return d
}
