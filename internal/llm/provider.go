// Package llm defines the FluxLens LLM provider abstraction.
//
// Every LLM integration in FluxLens (OpenAI, Ollama, vLLM, mock) implements
// the Provider interface. The orchestrator never speaks to a concrete LLM
// SDK directly; it speaks to a Provider. This keeps the orchestrator's
// guardrails and audit logic completely independent of provider-specific
// API quirks and makes per-environment provider selection a configuration
// concern rather than a code-change concern.
package llm

import "context"

// Provider is the minimum surface FluxLens needs from an LLM backend.
type Provider interface {
	// Name returns a stable identifier (e.g., "openai", "ollama", "mock").
	Name() string

	// ModelID returns the specific model the provider is configured with.
	ModelID() string

	// Decide invokes the model on the given request and returns a parsed
	// Decision response. Provider implementations are responsible for
	// prompt construction, network calls, retries, and timeouts.
	Decide(ctx context.Context, req DecisionRequest) (DecisionResponse, error)

	// Close releases any resources held by the provider.
	Close() error
}

// DecisionRequest is the input to an LLM decision call.
type DecisionRequest struct {
	// Context is operator-supplied domain context (event payload, recent
	// history, role hints).
	Context string

	// Instruction is the operator-configured instruction template.
	Instruction string

	// Examples are few-shot examples (optional).
	Examples []Example
}

// Example is a single few-shot example.
type Example struct {
	Input  string
	Output string
}

// DecisionResponse is the parsed and validated LLM output.
//
// FluxLens always requires its providers to return responses parsable into
// this shape; providers are responsible for asking the model in a way that
// produces a parseable response (typically JSON-mode or structured output).
type DecisionResponse struct {
	Classification string   `json:"classification"`
	Suggestion     string   `json:"suggestion"`
	Confidence     float64  `json:"confidence"`
	RequiresReview bool     `json:"requires_review"`
	Reasons        []string `json:"reasons,omitempty"`
}
