package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/llm"
)

// Orchestrator is the FluxLens AI decision-support layer. It validates
// input through Guardrails, calls the configured LLM Provider, validates
// the response through Guardrails again, writes a tamper-evident audit
// record, and returns a Decision the caller can surface to an operator.
//
// The orchestrator NEVER takes a consequential action on its own; the
// returned Decision is a suggestion. The caller is responsible for
// surfacing it to a human and recording the operator's action.
type Orchestrator struct {
	provider   llm.Provider
	guardrails *Guardrails
	chain      *auditlog.Chain
	tenant     string
}

// New returns a configured Orchestrator.
func New(provider llm.Provider, guardrails *Guardrails, chain *auditlog.Chain, tenant string) *Orchestrator {
	if guardrails == nil {
		guardrails = NewDefaultGuardrails()
	}
	if chain == nil {
		chain = auditlog.NewChain()
	}
	if tenant == "" {
		tenant = "default"
	}
	return &Orchestrator{provider: provider, guardrails: guardrails, chain: chain, tenant: tenant}
}

// Decision is the orchestrator's per-event result.
type Decision struct {
	EventID         string
	Provider        string
	ModelID         string
	PromptHash      string
	Response        Response
	Guardrails      string // "pass" | "rejected_input" | "rejected_output" | "provider_error"
	OperatorReview  bool   // true if requires_review is set or guardrails marked review
	AuditChainHash  string
	AuditChainPrev  string
}

// Decide runs the full decision pipeline on a single canonical event.
// The instruction template is supplied by the caller (per role or per
// domain pack).
func (o *Orchestrator) Decide(ctx context.Context, event canonical.Event, instruction string) (Decision, error) {
	if o.provider == nil {
		return Decision{}, errors.New("orchestrator: no provider configured")
	}
	if err := event.Validate(); err != nil {
		return Decision{}, fmt.Errorf("orchestrator: invalid event: %w", err)
	}

	contextBytes, _ := json.Marshal(map[string]any{
		"event_id":    event.EventID,
		"source_id":   event.SourceID,
		"event_type":  event.EventType,
		"severity":    event.Severity,
		"timestamp":   event.Timestamp,
		"payload":     event.Payload,
	})
	contextStr := string(contextBytes)

	if err := o.guardrails.ValidateInput(contextStr + "\n" + instruction); err != nil {
		dec := Decision{EventID: event.EventID, Provider: o.provider.Name(), ModelID: o.provider.ModelID(), Guardrails: "rejected_input", OperatorReview: true}
		dec.PromptHash = promptHash(contextStr, instruction)
		dec.AuditChainPrev = o.chain.HeadHash()
		rec, _ := o.chain.Append("decision_rejected_input", map[string]any{
			"event_id":    event.EventID,
			"reason":      err.Error(),
			"provider":    o.provider.Name(),
			"model_id":    o.provider.ModelID(),
			"prompt_hash": dec.PromptHash,
			"tenant":      o.tenant,
		})
		dec.AuditChainHash = rec.Hash
		return dec, nil
	}

	req := llm.DecisionRequest{Context: contextStr, Instruction: instruction}
	llmResp, err := o.provider.Decide(ctx, req)
	if err != nil {
		dec := Decision{EventID: event.EventID, Provider: o.provider.Name(), ModelID: o.provider.ModelID(), Guardrails: "provider_error", OperatorReview: true}
		dec.PromptHash = promptHash(contextStr, instruction)
		dec.AuditChainPrev = o.chain.HeadHash()
		rec, _ := o.chain.Append("decision_provider_error", map[string]any{
			"event_id":    event.EventID,
			"error":       err.Error(),
			"provider":    o.provider.Name(),
			"model_id":    o.provider.ModelID(),
			"prompt_hash": dec.PromptHash,
			"tenant":      o.tenant,
		})
		dec.AuditChainHash = rec.Hash
		return dec, nil
	}

	resp := Response{
		Classification: llmResp.Classification,
		Suggestion:     llmResp.Suggestion,
		Confidence:     llmResp.Confidence,
		RequiresReview: llmResp.RequiresReview,
		Reasons:        llmResp.Reasons,
	}
	if err := o.guardrails.ValidateOutput(resp); err != nil {
		dec := Decision{EventID: event.EventID, Provider: o.provider.Name(), ModelID: o.provider.ModelID(), Response: resp, Guardrails: "rejected_output", OperatorReview: true}
		dec.PromptHash = promptHash(contextStr, instruction)
		dec.AuditChainPrev = o.chain.HeadHash()
		rec, _ := o.chain.Append("decision_rejected_output", map[string]any{
			"event_id":    event.EventID,
			"reason":      err.Error(),
			"response":    resp,
			"provider":    o.provider.Name(),
			"model_id":    o.provider.ModelID(),
			"prompt_hash": dec.PromptHash,
			"tenant":      o.tenant,
		})
		dec.AuditChainHash = rec.Hash
		return dec, nil
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
	rec, _ := o.chain.Append("decision", map[string]any{
		"event_id":          event.EventID,
		"provider":          o.provider.Name(),
		"model_id":          o.provider.ModelID(),
		"prompt_hash":       dec.PromptHash,
		"response":          resp,
		"guardrails_status": dec.Guardrails,
		"operator_review":   dec.OperatorReview,
		"tenant":            o.tenant,
	})
	dec.AuditChainHash = rec.Hash
	return dec, nil
}

// RecordOperatorAction appends a tamper-evident audit record for an
// operator's accept/override/annotate action on a prior decision.
func (o *Orchestrator) RecordOperatorAction(decisionID, operatorID, action, annotation string) (string, error) {
	if action != "accept" && action != "override" && action != "annotate" {
		return "", fmt.Errorf("orchestrator: invalid operator action %q", action)
	}
	rec, err := o.chain.Append("operator_action", map[string]any{
		"decision_id": decisionID,
		"operator_id": operatorID,
		"action":      action,
		"annotation":  annotation,
		"tenant":      o.tenant,
	})
	if err != nil {
		return "", err
	}
	return rec.Hash, nil
}

func promptHash(ctx, instruction string) string {
	h := sha256.Sum256([]byte(ctx + "\x00" + instruction))
	return hex.EncodeToString(h[:])
}
