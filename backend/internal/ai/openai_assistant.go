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

// OpenAIAssistantProvider implements AssistantProvider using OpenAI-compatible API.
type OpenAIAssistantProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAIAssistantProvider(baseURL, apiKey, model string) *OpenAIAssistantProvider {
	return &OpenAIAssistantProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (p *OpenAIAssistantProvider) chat(ctx context.Context, system, user string, maxTokens int) (string, error) {
	req := openAIReq{
		Model: p.model,
		Messages: []openAIMsg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens: maxTokens,
	}
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai chat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai chat: status %d", resp.StatusCode)
	}

	var apiResp openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("openai chat decode: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("openai chat: no choices")
	}
	return strings.TrimSpace(apiResp.Choices[0].Message.Content), nil
}

func (p *OpenAIAssistantProvider) GoalToProofs(ctx context.Context, goalText string) (GoalToProofsResult, error) {
	system := `Ты помощник по постановке целей. По описанию цели предложи 3-5 конкретных proof contract варианта.
Каждый вариант должен быть: конкретным, проверяемым, с артефактом.
Отвечай ТОЛЬКО валидным JSON без пояснений: {"suggestions":[{"what_to_prove":"...","how_to_prove":"...","movement_mode":"single_proof|regular_rhythm|challenge","due_days":7}]}`

	raw, err := p.chat(ctx, system, goalText, 800)
	if err != nil {
		return GoalToProofsResult{}, err
	}

	var result GoalToProofsResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return GoalToProofsResult{}, fmt.Errorf("goal-to-proofs parse: %w", err)
	}
	return result, nil
}

func (p *OpenAIAssistantProvider) NextStep(ctx context.Context, goalText string, recentActivity string) (NextStepResult, error) {
	system := `Ты коуч по продуктивности. На основе цели и недавней активности предложи ОДИН маленький конкретный шаг на эту неделю.
Шаг должен занимать 1-3 часа, иметь конкретный результат.
Отвечай ТОЛЬКО валидным JSON: {"step":"...","rationale":"...","due_days":7}`

	userMsg := fmt.Sprintf("Цель: %s\nНедавняя активность: %s", goalText, recentActivity)
	raw, err := p.chat(ctx, system, userMsg, 300)
	if err != nil {
		return NextStepResult{}, err
	}

	var result NextStepResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return NextStepResult{}, fmt.Errorf("next-step parse: %w", err)
	}
	return result, nil
}

func (p *OpenAIAssistantProvider) ProofCheck(ctx context.Context, draft string) (ProofCheckResult, error) {
	system := `Ты проверяешь качество пруфа (доказательства выполненной работы).
Хороший пруф содержит: 1) конкретный артефакт (ссылка, скриншот, файл), 2) вывод что удалось понять/сделать, 3) следующий шаг.
Оцени черновик по этим критериям.
Отвечай ТОЛЬКО валидным JSON: {"has_artifact":true/false,"has_conclusion":true/false,"has_next_step":true/false,"score":0-100,"feedback":"...","suggestion":"..."}`

	raw, err := p.chat(ctx, system, draft, 400)
	if err != nil {
		return ProofCheckResult{}, err
	}

	var result ProofCheckResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return ProofCheckResult{}, fmt.Errorf("proof-check parse: %w", err)
	}
	return result, nil
}

func (p *OpenAIAssistantProvider) AntiProof(ctx context.Context, goalText, stuckDescription string) (AntiProofResult, error) {
	system := `Ты помогаешь пользователю оформить "анти-пруф" — честное описание того что не получилось.
Анти-пруф ценен: он фиксирует попытку, блокер и следующий шаг. Это НЕ провал, это данные.
Сформируй краткое описание попытки на основе ввода пользователя.
Отвечай ТОЛЬКО валидным JSON: {"summary":"...","what_blocked":"...","next_attempt":"..."}`

	userMsg := fmt.Sprintf("Цель: %s\nЧто случилось: %s", goalText, stuckDescription)
	raw, err := p.chat(ctx, system, userMsg, 400)
	if err != nil {
		return AntiProofResult{}, err
	}

	var result AntiProofResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return AntiProofResult{}, fmt.Errorf("anti-proof parse: %w", err)
	}
	return result, nil
}

func (p *OpenAIAssistantProvider) GrowthDossier(ctx context.Context, inputs []DossierInput) (GrowthDossierResult, error) {
	system := `Ты генерируешь досье роста — структурированный итог периода для встречи 1:1 или ИПР.
Напиши конкретно, без воды. Выдели навыки, достижения и следующий фокус.
Отвечай ТОЛЬКО валидным JSON: {"period_label":"...","total_proofs":0,"active_weeks":0,"top_skills":["..."],"goals":[{"goal_title":"...","proofs_count":0,"highlights":["..."],"skills":["..."],"conclusion":"..."}],"overall_summary":"...","next_focus":"..."}`

	var sb strings.Builder
	for _, g := range inputs {
		sb.WriteString(fmt.Sprintf("Цель: %s\n", g.GoalTitle))
		sb.WriteString(fmt.Sprintf("Период: %s — %s\n", g.StartDate, g.EndDate))
		for i, pt := range g.ProofsTexts {
			sb.WriteString(fmt.Sprintf("Пруф %d: %s\n", i+1, pt))
		}
		sb.WriteString("\n")
	}

	raw, err := p.chat(ctx, system, sb.String(), 1200)
	if err != nil {
		return GrowthDossierResult{}, err
	}

	var result GrowthDossierResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return GrowthDossierResult{}, fmt.Errorf("dossier parse: %w", err)
	}
	return result, nil
}
