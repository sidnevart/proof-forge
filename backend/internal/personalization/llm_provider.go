package personalization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LLMProvider talks to an Ollama-compatible endpoint.
type LLMProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewLLMProvider builds a provider.
func NewLLMProvider(baseURL, apiKey, model string) *LLMProvider {
	return &LLMProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// PromptResult is the raw LLM output.
type PromptResult struct {
	Text      string
	TokensIn  int64
	TokensOut int64
}

// Prompt sends a single-turn completion request.
func (p *LLMProvider) Prompt(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (PromptResult, error) {
	body := map[string]any{
		"model":  p.model,
		"prompt": userPrompt,
		"system": systemPrompt,
		"options": map[string]any{
			"num_predict": maxTokens,
		},
		"stream": false,
	}
	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return PromptResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return PromptResult{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PromptResult{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
		PromptTokens    int64 `json:"prompt_eval_count"`
		CompletionTokens int64 `json:"eval_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return PromptResult{}, fmt.Errorf("decode response: %w", err)
	}

	return PromptResult{
		Text:      result.Response,
		TokensIn:  result.PromptTokens,
		TokensOut: result.CompletionTokens,
	}, nil
}
