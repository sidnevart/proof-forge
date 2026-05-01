package recaps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *OpenAIProvider) Summarize(ctx context.Context, prompt string) (string, string, error) {
	payload := openAIRequest{
		Model:     p.model,
		Messages:  []openAIMessage{{Role: "user", Content: prompt}},
		MaxTokens: 500,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", "", fmt.Errorf("openai returned no choices")
	}
	return result.Choices[0].Message.Content, result.Model, nil
}
