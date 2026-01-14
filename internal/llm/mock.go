package llm

import (
	"context"
	"errors"
	"sync/atomic"
)

// MockProvider is an LLM provider for tests and demos. It returns a
// configurable canned response (or a sequence of responses) and counts
// invocations.
type MockProvider struct {
	modelID    string
	responses  []DecisionResponse
	failNext   atomic.Bool
	calls      atomic.Uint64
	cursor     atomic.Uint64
	failureErr error
}

// NewMockProvider returns a mock that cycles through the given responses.
// If responses is empty, a default routine-classification response is used.
func NewMockProvider(modelID string, responses ...DecisionResponse) *MockProvider {
	if len(responses) == 0 {
		responses = []DecisionResponse{
			{Classification: "routine", Suggestion: "no operator action required", Confidence: 0.85, RequiresReview: false},
		}
	}
	return &MockProvider{modelID: modelID, responses: responses, failureErr: errors.New("mock: injected failure")}
}

// Name implements Provider.
func (m *MockProvider) Name() string { return "mock" }

// ModelID implements Provider.
func (m *MockProvider) ModelID() string { return m.modelID }

// Decide implements Provider.
func (m *MockProvider) Decide(_ context.Context, _ DecisionRequest) (DecisionResponse, error) {
	m.calls.Add(1)
	if m.failNext.Load() {
		m.failNext.Store(false)
		return DecisionResponse{}, m.failureErr
	}
	idx := int(m.cursor.Add(1)-1) % len(m.responses)
	return m.responses[idx], nil
}

// Close implements Provider.
func (m *MockProvider) Close() error { return nil }

// FailNextCall makes the next Decide call return an error.
func (m *MockProvider) FailNextCall() { m.failNext.Store(true) }

// Calls returns the total number of Decide invocations made.
func (m *MockProvider) Calls() uint64 { return m.calls.Load() }
