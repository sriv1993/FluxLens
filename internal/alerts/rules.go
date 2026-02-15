package alerts

import (
	crand "crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/sriharshav1/fluxlens/internal/canonical"
)

func newID() string {
	id, err := ulid.New(ulid.Now(), crand.Reader)
	if err != nil {
		return fmt.Sprintf("alt-%d", time.Now().UnixNano())
	}
	return id.String()
}

// FromIngestEvent raises alerts when canonical severity crosses the warning line.
func FromIngestEvent(e canonical.Event) []Alert {
	if e.Severity.Rank() < canonical.SeverityWarn.Rank() {
		return nil
	}
	sev := Severity(e.Severity)
	title := fmt.Sprintf("Elevated ingest severity (%s)", e.EventType)
	body := fmt.Sprintf("event_id=%s source_id=%s severity=%s trace=%s",
		e.EventID, e.SourceID, e.Severity, e.Metadata.TraceID)
	return []Alert{{
		ID:        newID(),
		Title:     title,
		Body:      body,
		Severity:  sev,
		RuleID:    "ingest.severity_warn_or_above",
		CreatedAt: time.Now().UTC(),
		Ref: map[string]string{
			"event_id":   e.EventID,
			"source_id":  e.SourceID,
			"event_type": e.EventType,
		},
	}}
}

// FromDigestQuality emits informational / warning alerts when curation scores drift.
func FromDigestQuality(freshness, diversity float64, strategy int, k, selected int) []Alert {
	var out []Alert
	if freshness < 0.22 {
		out = append(out, Alert{
			ID:        newID(),
			Title:     "Digest freshness below threshold",
			Body:      fmt.Sprintf("freshness=%.3f strategy=%d k=%d selected=%d", freshness, strategy, k, selected),
			Severity:  SeverityWarn,
			RuleID:    "digest.low_freshness",
			CreatedAt: time.Now().UTC(),
			Ref: map[string]string{
				"freshness": fmt.Sprintf("%.4f", freshness),
				"strategy":  fmt.Sprintf("%d", strategy),
			},
		})
	}
	if diversity < 0.38 {
		out = append(out, Alert{
			ID:        newID(),
			Title:     "Digest diversity below threshold",
			Body:      fmt.Sprintf("diversity=%.3f strategy=%d k=%d selected=%d", diversity, strategy, k, selected),
			Severity:  SeverityInfo,
			RuleID:    "digest.low_diversity",
			CreatedAt: time.Now().UTC(),
			Ref: map[string]string{
				"diversity": fmt.Sprintf("%.4f", diversity),
				"strategy":  fmt.Sprintf("%d", strategy),
			},
		})
	}
	if k > 4 && selected > 0 && selected < k/3 {
		out = append(out, Alert{
			ID:        newID(),
			Title:     "Digest selection unexpectedly sparse",
			Body:      fmt.Sprintf("requested k=%d but only %d events matched filters/strategy", k, selected),
			Severity:  SeverityWarn,
			RuleID:    "digest.sparse_selection",
			CreatedAt: time.Now().UTC(),
			Ref: map[string]string{
				"k":        fmt.Sprintf("%d", k),
				"selected": fmt.Sprintf("%d", selected),
			},
		})
	}
	return out
}

// FromOperatorReviewRequired fires when AI output requires human review.
func FromOperatorReviewRequired(eventID, classification string) []Alert {
	return []Alert{{
		ID:        newID(),
		Title:     "AI suggestion requires operator review",
		Body:      fmt.Sprintf("event_id=%s classification=%s — acknowledge via operator resolve endpoint.", eventID, classification),
		Severity:  SeverityWarn,
		RuleID:    "operator.review_required",
		CreatedAt: time.Now().UTC(),
		Ref: map[string]string{
			"event_id":       eventID,
			"classification": classification,
		},
	}}
}

// FromOperatorResolution confirms an audited human decision on the suggestion.
func FromOperatorResolution(eventID, operatorID, action string) []Alert {
	sev := SeverityInfo
	if action == "override" {
		sev = SeverityWarn
	}
	return []Alert{{
		ID:        newID(),
		Title:     fmt.Sprintf("Operator recorded %s", action),
		Body:      fmt.Sprintf("operator_id=%s event_id=%s — tamper-evident audit append succeeded.", operatorID, eventID),
		Severity:  sev,
		RuleID:    "operator.resolution_recorded",
		CreatedAt: time.Now().UTC(),
		Ref: map[string]string{
			"event_id":    eventID,
			"operator_id": operatorID,
			"action":      action,
		},
	}}
}

// FromAuditBroken returns a critical alert when the audit chain transitions from valid to invalid.
func FromAuditBroken(verifyErr error) []Alert {
	msg := "audit chain verification failed"
	if verifyErr != nil {
		msg = verifyErr.Error()
	}
	return []Alert{{
		ID:        newID(),
		Title:     "Audit chain integrity failure",
		Body:      msg,
		Severity:  SeverityCritical,
		RuleID:    "platform.audit_chain_invalid",
		CreatedAt: time.Now().UTC(),
	}}
}
