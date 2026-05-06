package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// EmbedProvider computes dense vector embeddings for text.
type EmbedProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// OpenAIEmbedProvider uses text-embedding-3-small to produce 1536-dim vectors.
type OpenAIEmbedProvider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAIEmbedProvider(baseURL, apiKey string) *OpenAIEmbedProvider {
	return &OpenAIEmbedProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (p *OpenAIEmbedProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(embedRequest{Model: "text-embedding-3-small", Input: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embed: status %d", resp.StatusCode)
	}
	var apiResp embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("openai embed: decode: %w", err)
	}
	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("openai embed: no data")
	}
	return apiResp.Data[0].Embedding, nil
}

// NoopEmbedProvider returns a zero vector — used when AI is disabled.
type NoopEmbedProvider struct{}

func (NoopEmbedProvider) Embed(_ context.Context, _ string) ([]float32, error) {
	return make([]float32, 1536), nil
}

type GoalRefineVariant struct {
	Title         string   `json:"title"`
	Smart         string   `json:"smart"`
	ProofExamples []string `json:"proof_examples"`
}

type GoalRefineResult struct {
	Category string              `json:"category"`
	Variants []GoalRefineVariant `json:"variants"`
}

type RefineProvider interface {
	RefineGoal(ctx context.Context, draftText string) (GoalRefineResult, error)
}

type FakeGoalRefineProvider struct{}

func NewFakeGoalRefineProvider() *FakeGoalRefineProvider {
	return &FakeGoalRefineProvider{}
}

func (p *FakeGoalRefineProvider) RefineGoal(_ context.Context, draftText string) (GoalRefineResult, error) {
	trimmed := strings.TrimSpace(draftText)
	normalized := strings.ToLower(trimmed)
	category := classifyCategory(normalized)

	return GoalRefineResult{
		Category: category,
		Variants: []GoalRefineVariant{
			buildOutcomeVariant(category, trimmed),
			buildCadenceVariant(category, trimmed),
			buildEvidenceVariant(category, trimmed),
		},
	}, nil
}

func classifyCategory(normalized string) string {
	switch {
	case containsAny(normalized, "англ", "учёб", "курс", "экзам"):
		return "учёба"
	case containsAny(normalized, "трен", "бег", "зал", "вес"):
		return "фитнес"
	case containsAny(normalized, "код", "релиз", "лендинг", "продукт", "проект"):
		return "работа"
	case containsAny(normalized, "рис", "музык", "пиш", "твор"):
		return "творчество"
	default:
		return "развитие"
	}
}

func buildOutcomeVariant(category, normalized string) GoalRefineVariant {
	return GoalRefineVariant{
		Title: "Сфокусировать результат",
		Smart: "Категория: " + category + ". За 4 недели довести цель \"" + normalized + "\" до измеримого результата с понятным критерием завершения.",
		ProofExamples: []string{
			"Скриншот или ссылка с итоговым результатом по цели \"" + normalized + "\".",
			"Короткий текстовый отчёт с числом выполненных шагов за неделю.",
			"Подтверждение от buddy, что результат по \"" + normalized + "\" достигнут.",
		},
	}
}

func buildCadenceVariant(category, normalized string) GoalRefineVariant {
	return GoalRefineVariant{
		Title: "Зафиксировать ритм",
		Smart: "Категория: " + category + ". В течение 3 недель выполнять цель \"" + normalized + "\" по расписанию 3 раза в неделю и не пропускать две сессии подряд.",
		ProofExamples: []string{
			"Запись в календаре или трекере с датами трёх сессий по \"" + normalized + "\".",
			"Фото, заметка или скриншот после каждой сессии по цели.",
			"Недельный итог с количеством выполненных сессий и выводом по прогрессу.",
		},
	}
}

func buildEvidenceVariant(category, normalized string) GoalRefineVariant {
	return GoalRefineVariant{
		Title: "Усилить доказательство",
		Smart: "Категория: " + category + ". За 2 недели собрать 6 подтверждений прогресса по цели \"" + normalized + "\", чтобы прогресс можно было проверить без дополнительных пояснений.",
		ProofExamples: []string{
			"Папка или список из шести артефактов, связанных с \"" + normalized + "\".",
			"Два сравнительных доказательства формата было/стало по цели.",
			"Один итоговый апдейт с перечислением всех собранных подтверждений.",
		},
	}
}

func containsAny(text string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}

const refineSystemPrompt = `Ты помогаешь сделать цель SMART. Верни ТОЛЬКО JSON без markdown-обёртки в формате:
{"category":"<одно слово: фитнес/учёба/работа/творчество/развитие>","variants":[{"title":"<3-5 слов>","smart":"<SMART-формулировка>","proof_examples":["<пример пруфа 1>","<пример пруфа 2>","<пример пруфа 3>"]}]}
Ровно 3 варианта: outcome-focused, cadence-focused, evidence-focused.`

// OpenAIRefineProvider calls GPT-4o-mini to produce SMART goal variants.
type OpenAIRefineProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAIRefineProvider(baseURL, apiKey, model string) *OpenAIRefineProvider {
	return &OpenAIRefineProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type openAIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIReq struct {
	Model     string      `json:"model"`
	Messages  []openAIMsg `json:"messages"`
	MaxTokens int         `json:"max_tokens"`
}

type openAIResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *OpenAIRefineProvider) RefineGoal(ctx context.Context, draftText string) (GoalRefineResult, error) {
	req := openAIReq{
		Model: p.model,
		Messages: []openAIMsg{
			{Role: "system", Content: refineSystemPrompt},
			{Role: "user", Content: draftText},
		},
		MaxTokens: 600,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return GoalRefineResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return GoalRefineResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return GoalRefineResult{}, fmt.Errorf("openai refine: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GoalRefineResult{}, fmt.Errorf("openai refine: status %d", resp.StatusCode)
	}

	var apiResp openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return GoalRefineResult{}, fmt.Errorf("openai refine: decode: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return GoalRefineResult{}, fmt.Errorf("openai refine: no choices")
	}

	raw := strings.TrimSpace(apiResp.Choices[0].Message.Content)
	var result GoalRefineResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return GoalRefineResult{}, fmt.Errorf("openai refine: parse json: %w", err)
	}
	return result, nil
}
