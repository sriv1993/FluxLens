package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/llm"
	"github.com/sriharshav1/fluxlens/internal/precedents"
)

// DefaultPrecedentInstruction is used when the caller omits a custom instruction.
const DefaultPrecedentInstruction = `You are assisting a manufacturing line supervisor.
Similar past operator resolutions are provided as precedents.
Classify the current event (routine / elevated / critical) and propose concrete operator steps.
Never recommend autonomous operational moves; bias toward human review when evidence is ambiguous.`

// PrecedentStep is one suggested action for the operator UI.
type PrecedentStep struct {
	Text               string `json:"text"`
	CitedPrecedentHash string `json:"cited_precedent_hash,omitempty"`
}

// PrecedentSuggestion bundles precedent-informed steps with the audited decision.
type PrecedentSuggestion struct {
	Steps          []PrecedentStep       `json:"steps"`
	PrecedentsUsed []precedents.Resolved `json:"precedents_used"`
	Decision       Decision              `json:"decision"`
}

// SuggestWithPrecedents retrieves matching past resolutions, calls the LLM with
// precedent context, appends a decision_with_precedents audit record on success,
// and returns human-reviewable steps (no autonomous actions).
func (o *Orchestrator) SuggestWithPrecedents(ctx context.Context, event canonical.Event, instruction string, maxPrecedents int) (PrecedentSuggestion, error) {
	if o.provider == nil {
		return PrecedentSuggestion{}, errors.New("orchestrator: no provider configured")
	}
	if err := event.Validate(); err != nil {
		return PrecedentSuggestion{}, fmt.Errorf("orchestrator: invalid event: %w", err)
	}
	if instruction == "" {
		instruction = DefaultPrecedentInstruction
	}

	tenant := eventTenant(event, o.tenant)
	crit := precedents.Criteria{
		EventType: event.EventType,
		SourceID:  event.SourceID,
		Severity:  string(event.Severity),
		TenantID:  tenant,
	}
	matched := precedents.FindMatches(o.chain, crit, maxPrecedents)

	contextBytes, _ := json.Marshal(map[string]any{
		"event_id":   event.EventID,
		"source_id":  event.SourceID,
		"event_type": event.EventType,
		"severity":   event.Severity,
		"timestamp":  event.Timestamp,
		"payload":    event.Payload,
		"precedents": matched,
	})
	contextStr := string(contextBytes)

	if err := o.guardrails.ValidateInput(contextStr + "\n" + instruction); err != nil {
		dec := o.auditPrecedentRejected(event, instruction, contextStr, "rejected_input", err.Error(), nil, matched)
		return PrecedentSuggestion{Steps: stepsFromRejected(dec), PrecedentsUsed: matched, Decision: dec}, nil
	}

	req := llm.DecisionRequest{Context: contextStr, Instruction: instruction}
	llmResp, err := o.provider.Decide(ctx, req)
	if err != nil {
		dec := o.auditPrecedentRejected(event, instruction, contextStr, "provider_error", err.Error(), nil, matched)
		return PrecedentSuggestion{Steps: stepsFromRejected(dec), PrecedentsUsed: matched, Decision: dec}, nil
	}

	resp := Response{
		Classification: llmResp.Classification,
		Suggestion:     llmResp.Suggestion,
		Confidence:     llmResp.Confidence,
		RequiresReview: llmResp.RequiresReview,
		Reasons:        llmResp.Reasons,
	}
	if err := o.guardrails.ValidateOutput(resp); err != nil {
		dec := o.auditPrecedentRejected(event, instruction, contextStr, "rejected_output", err.Error(), &resp, matched)
		return PrecedentSuggestion{Steps: stepsFromRejected(dec), PrecedentsUsed: matched, Decision: dec}, nil
	}

	dec := Decision{
		EventID:        event.EventID,
		Provider:       o.provider.Name(),
		ModelID:        o.provider.ModelID(),
		PromptHash:     promptHash(contextStr, instruction),
		Response:       resp,
		Guardrails:     "pass",
		OperatorReview: resp.RequiresReview,
		AuditChainPrev: o.chain.HeadHash(),
	}
	rec, _ := o.chain.Append("decision_with_precedents", map[string]any{
		"event_id":          event.EventID,
		"event_type":        event.EventType,
		"source_id":         event.SourceID,
		"severity":          string(event.Severity),
		"tenant":            tenant,
		"provider":          o.provider.Name(),
		"model_id":          o.provider.ModelID(),
		"prompt_hash":       dec.PromptHash,
		"response":          resp,
		"guardrails_status": dec.Guardrails,
		"operator_review":   dec.OperatorReview,
		"precedent_count":   len(matched),
		"precedent_hashes":  precedentHashes(matched),
	})
	dec.AuditChainHash = rec.Hash

	steps := BuildSteps(resp, matched)
	return PrecedentSuggestion{Steps: steps, PrecedentsUsed: matched, Decision: dec}, nil
}

// BuildSteps turns the LLM response and matched precedents into operator-facing steps.
func BuildSteps(resp Response, matched []precedents.Resolved) []PrecedentStep {
	steps := []PrecedentStep{{
		Text: fmt.Sprintf("%s — %s", resp.Classification, resp.Suggestion),
	}}
	for _, p := range matched {
		text := fmt.Sprintf("Prior %s on similar %s/%s: operator %s",
			p.Classification, p.EventType, p.Severity, p.OperatorAction)
		if p.Annotation != "" {
			text += " — " + p.Annotation
		}
		if p.Suggestion != "" {
			text += fmt.Sprintf(" (AI had suggested: %s)", p.Suggestion)
		}
		steps = append(steps, PrecedentStep{
			Text:               text,
			CitedPrecedentHash: p.DecisionHash,
		})
	}
	return steps
}

func stepsFromRejected(dec Decision) []PrecedentStep {
	msg := dec.Guardrails
	if dec.Response.Suggestion != "" {
		msg = dec.Response.Suggestion
	}
	return []PrecedentStep{{Text: "Suggestion blocked by guardrails (" + msg + "); review manually."}}
}

func (o *Orchestrator) auditPrecedentRejected(event canonical.Event, instruction, contextStr, status, reason string, resp *Response, matched []precedents.Resolved) Decision {
	tenant := eventTenant(event, o.tenant)
	dec := Decision{
		EventID:        event.EventID,
		Provider:       o.provider.Name(),
		ModelID:        o.provider.ModelID(),
		PromptHash:     promptHash(contextStr, instruction),
		Guardrails:     status,
		OperatorReview: true,
		AuditChainPrev: o.chain.HeadHash(),
	}
	if resp != nil {
		dec.Response = *resp
	}
	payload := map[string]any{
		"event_id":          event.EventID,
		"event_type":        event.EventType,
		"source_id":         event.SourceID,
		"severity":          string(event.Severity),
		"tenant":            tenant,
		"reason":            reason,
		"provider":          o.provider.Name(),
		"model_id":          o.provider.ModelID(),
		"prompt_hash":       dec.PromptHash,
		"guardrails_status": status,
		"precedent_count":   len(matched),
	}
	if resp != nil {
		payload["response"] = resp
	}
	rec, _ := o.chain.Append("decision_"+status, payload)
	dec.AuditChainHash = rec.Hash
	return dec
}

func eventTenant(event canonical.Event, fallback string) string {
	if event.Metadata.TenantID != "" {
		return event.Metadata.TenantID
	}
	if fallback != "" {
		return fallback
	}
	return "default"
}

func precedentHashes(m []precedents.Resolved) []string {
	out := make([]string, len(m))
	for i, p := range m {
		out[i] = p.DecisionHash
	}
	return out
}
