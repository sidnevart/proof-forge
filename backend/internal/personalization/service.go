package personalization

import (
	"context"
	"fmt"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
)

// Service is the personalization orchestrator.
type Service struct {
	enabled       bool
	budgetLimit   int64
	cbStore       CircuitBreakerStore
	budgetStore   BudgetStore
	llm           *LLMProvider
	template      *TemplateProvider
	recorder      analytics.Recorder
	clock         func() time.Time
}

// ServiceOption customises the service.
type ServiceOption func(*Service)

// WithBudgetLimit sets the daily token cap.
func WithBudgetLimit(limit int64) ServiceOption {
	return func(s *Service) { s.budgetLimit = limit }
}

// WithRecorder injects an analytics recorder.
func WithRecorder(r analytics.Recorder) ServiceOption {
	return func(s *Service) { s.recorder = r }
}

// NewService builds the personalization service.
func NewService(enabled bool, cbStore CircuitBreakerStore, budgetStore BudgetStore, llm *LLMProvider, opts ...ServiceOption) *Service {
	s := &Service{
		enabled:     enabled,
		budgetLimit: DefaultDailyBudgetTokens,
		cbStore:     cbStore,
		budgetStore: budgetStore,
		llm:         llm,
		template:    NewTemplateProvider(),
		clock:       time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Prompt is a convenience wrapper over Run that calls the configured LLM provider.
func (s *Service) Prompt(ctx context.Context, feature Feature, mode Mode, timeout time.Duration, systemPrompt, userPrompt string, maxTokens int) (Result, InvocationEvent, error) {
	return s.Run(ctx, feature, mode, timeout, func(ctx context.Context) (PromptResult, error) {
		if s.llm == nil {
			return PromptResult{}, fmt.Errorf("llm provider not configured")
		}
		return s.llm.Prompt(ctx, systemPrompt, userPrompt, maxTokens)
	})
}

// Run executes personalization with full fallback chain.
func (s *Service) Run(ctx context.Context, feature Feature, mode Mode, timeout time.Duration, llmPrompt func(ctx context.Context) (PromptResult, error)) (Result, InvocationEvent, error) {
	start := s.clock()
	ev := InvocationEvent{
		Feature: feature,
		Mode:    mode,
	}

	if !s.enabled {
		ev.Provider = ProviderTemplate
		ev.FallbackReason = "disabled globally"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return Result{Text: text, Provider: ProviderTemplate}, ev, nil
	}

	if mode == ModeOff {
		ev.Provider = ProviderTemplate
		ev.FallbackReason = "mode off"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return Result{Text: text, Provider: ProviderTemplate}, ev, nil
	}

	if err := CheckCircuit(ctx, s.cbStore, feature, start); err != nil {
		ev.Provider = ProviderTemplate
		ev.FallbackReason = "circuit open"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return Result{Text: text, Provider: ProviderTemplate}, ev, nil
	}

	if err := CheckBudget(ctx, s.budgetStore, start, s.budgetLimit); err != nil {
		ev.Provider = ProviderTemplate
		ev.FallbackReason = "budget exceeded"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return Result{Text: text, Provider: ProviderTemplate}, ev, nil
	}

	// Attempt LLM with timeout.
	llmCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := llmPrompt(llmCtx)
	latency := s.clock().Sub(start).Milliseconds()
	ev.LatencyMs = latency

	if err != nil {
		ev.Provider = ProviderTemplate
		ev.FallbackReason = "provider error"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return Result{Text: text, Provider: ProviderTemplate}, ev, nil
	}

	ev.Provider = ProviderKimi
	ev.TokensIn = res.TokensIn
	ev.TokensOut = res.TokensOut
	_ = s.budgetStore.Increment(ctx, start, res.TokensIn, res.TokensOut, false)

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventPersonalizationInvoked, analytics.SourceSystem, nil, nil, nil, nil, map[string]any{
			"feature": string(feature),
			"mode": string(mode),
			"provider": string(ev.Provider),
			"latency_ms": ev.LatencyMs,
			"tokens_in": ev.TokensIn,
			"tokens_out": ev.TokensOut,
			"fallback_reason": ev.FallbackReason,
			"prompt_version": ev.PromptVersion,
		})
		if ev.FallbackReason != "" {
			_ = s.recorder.Record(ctx, analytics.EventPersonalizationFallback, analytics.SourceSystem, nil, nil, nil, nil, map[string]any{
				"feature": string(feature),
				"mode": string(mode),
				"fallback_reason": ev.FallbackReason,
				"prompt_version": ev.PromptVersion,
			})
		}
	}

	return Result{Text: res.Text, Provider: ProviderKimi}, ev, nil
}

// ValidateEveningPing enforces the Phase 3 rules.
func ValidateEveningPing(text string) error {
	// 8-25 words, ends with ?, no forbidden words.
	words := len(splitWords(text))
	if words < 8 || words > 25 {
		return fmt.Errorf("%w: expected 8-25 words, got %d", ErrValidationFailed, words)
	}
	if len(text) > 0 && text[len(text)-1] != '?' {
		return fmt.Errorf("%w: must end with question mark", ErrValidationFailed)
	}
	for _, bad := range []string{"молодец", "отлично", "плохо", "ты должен"} {
		if contains(text, bad) {
			return fmt.Errorf("%w: contains forbidden word %q", ErrValidationFailed, bad)
		}
	}
	return nil
}

func splitWords(s string) []string {
	// Very simple split for MVP.
	var words []string
	start := -1
	for i, r := range s {
		if r > ' ' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			words = append(words, s[start:i])
			start = -1
		}
	}
	if start >= 0 {
		words = append(words, s[start:])
	}
	return words
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
