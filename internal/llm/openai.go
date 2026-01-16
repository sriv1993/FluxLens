package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider implements Provider against the OpenAI-compatible REST
// API (which Ollama, vLLM, and others also expose). For Phase 1 the
// provider supports the /v1/chat/completions endpoint with JSON-mode
// outputs.
//
// The endpoint URL is configurable so the same code talks to:
//   - OpenAI:  https://api.openai.com
//   - Ollama:  http://localhost:11434
//   - vLLM:    http://localhost:8000
//   - WireMock (dev/CI): http://localhost:8080
type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAIProvider returns a configured provider. baseURL should NOT
// include the trailing /v1; the path is appended by the provider.
func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements Provider.
func (o *OpenAIProvider) Name() string { return "openai-compatible" }

// ModelID implements Provider.
func (o *OpenAIProvider) ModelID() string { return o.model }

// Close implements Provider.
func (o *OpenAIProvider) Close() error { return nil }

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	N           int           `json:"n"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Decide implements Provider.
func (o *OpenAIProvider) Decide(ctx context.Context, req DecisionRequest) (DecisionResponse, error) {
	system := "You are an industrial operations decision-support model. You must respond ONLY with strict JSON matching the schema: " +
		`{"classification":"string","suggestion":"string","confidence":number 0-1,"requires_review":bool,"reasons":["string"]}. ` +
		"If you are uncertain, set requires_review to true and confidence to a value below 0.6."
	user := fmt.Sprintf("Instruction: %s\n\nContext:\n%s", req.Instruction, req.Context)

	body, err := json.Marshal(chatRequest{
		Model:       o.model,
		Messages:    []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature: 0.2,
		N:           1,
	})
	if err != nil {
		return DecisionResponse{}, fmt.Errorf("llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return DecisionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return DecisionResponse{}, fmt.Errorf("llm: http call: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return DecisionResponse{}, fmt.Errorf("llm: provider returned %d: %s", resp.StatusCode, string(raw))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return DecisionResponse{}, fmt.Errorf("llm: parse response envelope: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return DecisionResponse{}, errors.New("llm: empty choices in provider response")
	}

	content := parsed.Choices[0].Message.Content
	var decision DecisionResponse
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return DecisionResponse{}, fmt.Errorf("llm: parse model JSON output: %w", err)
	}
	return decision, nil
}
