package alerts

import (
	"time"
)

// Severity mirrors canonical severity strings for JSON stability with the dashboard.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Alert is an operator-visible notification derived from ingest, digest quality,
// or platform integrity signals.
type Alert struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Severity  Severity          `json:"severity"`
	RuleID    string            `json:"rule_id"`
	CreatedAt time.Time         `json:"created_at"`
	Ref       map[string]string `json:"ref,omitempty"`
}
