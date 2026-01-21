// Package orchestrator implements the FluxLens AI decision-support layer.
//
// Every interaction with a Large Language Model passes through this
// package's Guardrails (input validation, prompt-injection scan, output
// schema validation, refusal-on-uncertainty), the human-override hook
// (HumanOverride), and the audit writer. These three components are
// enforced in code, not in policy; bypassing them is not a configuration
// option.
package orchestrator

import (
	"errors"
	"regexp"
	"strings"
)

// Guardrails enforces input and output safety constraints around LLM
// calls. Operators may extend by registering additional Validators.
type Guardrails struct {
	maxInputBytes      int
	minConfidence      float64
	promptInjectionRgx []*regexp.Regexp
	inputValidators    []func(string) error
	outputValidators   []func(Response) error
}

// Response is the shape FluxLens requires from any LLM call.
type Response struct {
	Classification string   `json:"classification"`
	Suggestion     string   `json:"suggestion"`
	Confidence     float64  `json:"confidence"`
	RequiresReview bool     `json:"requires_review"`
	Reasons        []string `json:"reasons,omitempty"`
}

// NewDefaultGuardrails returns a Guardrails configured with FluxLens
// default policies. Operators can extend or replace via setters.
func NewDefaultGuardrails() *Guardrails {
	return &Guardrails{
		maxInputBytes: 65536,
		minConfidence: 0.6,
		promptInjectionRgx: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ignore (all )?previous instructions`),
			regexp.MustCompile(`(?i)disregard (the )?(system|developer) (prompt|instructions)`),
			regexp.MustCompile(`(?i)you are now (a |an )?[a-z]+ assistant`),
		},
	}
}

// ValidateInput is called before every LLM call. Returns nil if the
// input is safe to send to the model.
func (g *Guardrails) ValidateInput(s string) error {
	if len(s) > g.maxInputBytes {
		return errors.New("orchestrator: input exceeds max bytes")
	}
	lower := strings.ToLower(s)
	for _, rg := range g.promptInjectionRgx {
		if rg.MatchString(lower) {
			return errors.New("orchestrator: prompt-injection pattern detected")
		}
	}
	for _, v := range g.inputValidators {
		if err := v(s); err != nil {
			return err
		}
	}
	return nil
}

// ValidateOutput is called after every LLM response is parsed. It
// enforces schema correctness and the refusal-on-uncertainty rule.
func (g *Guardrails) ValidateOutput(r Response) error {
	if r.Classification == "" {
		return errors.New("orchestrator: classification required")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return errors.New("orchestrator: confidence out of range")
	}
	if r.Confidence < g.minConfidence {
		// Below confidence floor: do not surface a suggestion; flag for review.
		if !r.RequiresReview {
			return errors.New("orchestrator: low confidence without requires_review")
		}
	}
	for _, v := range g.outputValidators {
		if err := v(r); err != nil {
			return err
		}
	}
	return nil
}

// SetMinConfidence sets the confidence floor below which the orchestrator
// requires the response to be flagged for human review.
func (g *Guardrails) SetMinConfidence(c float64) { g.minConfidence = c }

// AddInputValidator registers a custom input validator.
func (g *Guardrails) AddInputValidator(v func(string) error) {
	g.inputValidators = append(g.inputValidators, v)
}

// AddOutputValidator registers a custom output validator.
func (g *Guardrails) AddOutputValidator(v func(Response) error) {
	g.outputValidators = append(g.outputValidators, v)
}
