package precedents

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
)

// Criteria identifies the event dimensions used to find similar past resolutions.
type Criteria struct {
	EventType string
	SourceID  string
	Severity  string
	TenantID  string
}

// Resolved is a past decision paired with an operator resolution from the audit chain.
type Resolved struct {
	DecisionHash      string `json:"decision_hash"`
	EventID           string `json:"event_id,omitempty"`
	EventType         string `json:"event_type"`
	SourceID          string `json:"source_id"`
	Severity          string `json:"severity"`
	Tenant            string `json:"tenant"`
	Classification    string `json:"classification,omitempty"`
	Suggestion        string `json:"suggestion,omitempty"`
	OperatorAction    string `json:"operator_action"`
	Annotation        string `json:"annotation,omitempty"`
	OperatorAuditHash string `json:"operator_audit_hash,omitempty"`
}

var decisionKinds = map[string]bool{
	"decision":                 true,
	"decision_with_precedents": true,
}

type decisionRow struct {
	hash           string
	eventID        string
	eventType      string
	sourceID       string
	severity       string
	tenant         string
	classification string
	suggestion     string
	sequence       uint64
}

type actionRow struct {
	decisionID string
	action     string
	annotation string
	hash       string
	sequence   uint64
}

// FindMatches scans the audit chain for prior operator resolutions that match
// all criteria dimensions (event_type, source_id, severity, tenant). Results
// are ordered newest-first and capped at max (default 5 when max <= 0).
func FindMatches(chain auditlog.Store, c Criteria, max int) []Resolved {
	if max <= 0 {
		max = 5
	}
	tenant := normalizeTenant(c.TenantID)

	decisions := map[string]decisionRow{}
	var actions []actionRow

	for _, rec := range chain.Snapshot() {
		switch {
		case decisionKinds[rec.Kind]:
			var p struct {
				EventID          string `json:"event_id"`
				EventType        string `json:"event_type"`
				SourceID         string `json:"source_id"`
				Severity         string `json:"severity"`
				Tenant           string `json:"tenant"`
				GuardrailsStatus string `json:"guardrails_status"`
				Response         *struct {
					Classification string `json:"classification"`
					Suggestion     string `json:"suggestion"`
				} `json:"response"`
			}
			if json.Unmarshal(rec.Payload, &p) != nil {
				continue
			}
			if p.GuardrailsStatus != "" && p.GuardrailsStatus != "pass" {
				continue
			}
			class, sugg := "", ""
			if p.Response != nil {
				class = p.Response.Classification
				sugg = p.Response.Suggestion
			}
			decisions[rec.Hash] = decisionRow{
				hash:           rec.Hash,
				eventID:        p.EventID,
				eventType:      p.EventType,
				sourceID:       p.SourceID,
				severity:       p.Severity,
				tenant:         normalizeTenant(p.Tenant),
				classification: class,
				suggestion:     sugg,
				sequence:       rec.Sequence,
			}
		case rec.Kind == "operator_action":
			var p struct {
				DecisionID string `json:"decision_id"`
				Action     string `json:"action"`
				Annotation string `json:"annotation"`
			}
			if json.Unmarshal(rec.Payload, &p) != nil || p.DecisionID == "" {
				continue
			}
			actions = append(actions, actionRow{
				decisionID: p.DecisionID,
				action:     p.Action,
				annotation: p.Annotation,
				hash:       rec.Hash,
				sequence:   rec.Sequence,
			})
		}
	}

	var out []Resolved
	for _, act := range actions {
		d, ok := decisions[act.decisionID]
		if !ok {
			continue
		}
		if !matchesCriteria(d, c, tenant) {
			continue
		}
		out = append(out, Resolved{
			DecisionHash:      d.hash,
			EventID:           d.eventID,
			EventType:         d.eventType,
			SourceID:          d.sourceID,
			Severity:          d.severity,
			Tenant:            d.tenant,
			Classification:    d.classification,
			Suggestion:        d.suggestion,
			OperatorAction:    act.action,
			Annotation:        act.annotation,
			OperatorAuditHash: act.hash,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return decisions[out[i].DecisionHash].sequence > decisions[out[j].DecisionHash].sequence
	})
	if len(out) > max {
		out = out[:max]
	}
	return out
}

func matchesCriteria(d decisionRow, c Criteria, tenant string) bool {
	if c.EventType == "" || c.SourceID == "" || c.Severity == "" {
		return false
	}
	if !strings.EqualFold(d.eventType, c.EventType) {
		return false
	}
	if d.sourceID != c.SourceID {
		return false
	}
	if !strings.EqualFold(d.severity, c.Severity) {
		return false
	}
	if tenant != "" && normalizeTenant(d.tenant) != tenant {
		return false
	}
	return true
}

func normalizeTenant(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return "default"
	}
	return t
}
