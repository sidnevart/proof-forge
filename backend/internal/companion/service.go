package companion

import (
	"context"
	"fmt"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
	"github.com/sidnevart/proof-forge/backend/internal/personalization"
)

// Service is the companion orchestrator.
type Service struct {
	enabled     bool
	budgetLimit int64
	cbStore     personalization.CircuitBreakerStore
	budgetStore personalization.BudgetStore
	repo        Repository
	llm         *personalization.LLMProvider
	template    *personalization.TemplateProvider
	recorder    analytics.Recorder
	clock       func() time.Time
}

// Option customises the service.
type Option func(*Service)

// WithBudgetLimit sets the daily token cap.
func WithBudgetLimit(limit int64) Option {
	return func(s *Service) { s.budgetLimit = limit }
}

// WithRecorder injects an analytics recorder.
func WithRecorder(r analytics.Recorder) Option {
	return func(s *Service) { s.recorder = r }
}

// NewService builds the companion service.
func NewService(enabled bool, cbStore personalization.CircuitBreakerStore, budgetStore personalization.BudgetStore, repo Repository, llm *personalization.LLMProvider, opts ...Option) *Service {
	s := &Service{
		enabled:     enabled,
		budgetLimit: personalization.DefaultDailyBudgetTokens,
		cbStore:     cbStore,
		budgetStore: budgetStore,
		repo:        repo,
		llm:         llm,
		recorder:    analytics.NoopRecorder{},
		clock:       time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run executes a companion feature with full fallback chain.
// It reuses the personalization circuit breaker and budget logic.
func (s *Service) Run(ctx context.Context, feature Feature, mode personalization.Mode, timeout time.Duration, llmPrompt func(ctx context.Context) (personalization.PromptResult, error)) (personalization.Result, personalization.InvocationEvent, error) {
	start := s.clock()
	ev := personalization.InvocationEvent{
		Feature: personalization.Feature(feature),
		Mode:    mode,
	}

	if !s.enabled {
		ev.Provider = personalization.ProviderTemplate
		ev.FallbackReason = "disabled globally"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return personalization.Result{Text: text, Provider: personalization.ProviderTemplate}, ev, nil
	}

	if mode == personalization.ModeOff {
		ev.Provider = personalization.ProviderTemplate
		ev.FallbackReason = "mode off"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return personalization.Result{Text: text, Provider: personalization.ProviderTemplate}, ev, nil
	}

	if err := personalization.CheckCircuit(ctx, s.cbStore, personalization.Feature(feature), start); err != nil {
		ev.Provider = personalization.ProviderTemplate
		ev.FallbackReason = "circuit open"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return personalization.Result{Text: text, Provider: personalization.ProviderTemplate}, ev, nil
	}

	if err := personalization.CheckBudget(ctx, s.budgetStore, start, s.budgetLimit); err != nil {
		ev.Provider = personalization.ProviderTemplate
		ev.FallbackReason = "budget exceeded"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return personalization.Result{Text: text, Provider: personalization.ProviderTemplate}, ev, nil
	}

	// Attempt LLM with timeout.
	llmCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := llmPrompt(llmCtx)
	latency := s.clock().Sub(start).Milliseconds()
	ev.LatencyMs = latency

	if err != nil {
		ev.Provider = personalization.ProviderTemplate
		ev.FallbackReason = "provider error"
		text, _ := s.template.EveningPing(ctx, 0, 0)
		return personalization.Result{Text: text, Provider: personalization.ProviderTemplate}, ev, nil
	}

	ev.Provider = personalization.ProviderKimi
	ev.TokensIn = res.TokensIn
	ev.TokensOut = res.TokensOut
	_ = s.budgetStore.Increment(ctx, start, res.TokensIn, res.TokensOut, false)

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventPersonalizationInvoked, analytics.SourceSystem, nil, nil, nil, nil, map[string]any{
			"feature":    string(feature),
			"mode":       string(mode),
			"provider":     string(ev.Provider),
			"latency_ms":   ev.LatencyMs,
			"tokens_in":    ev.TokensIn,
			"tokens_out":   ev.TokensOut,
			"fallback_reason": ev.FallbackReason,
		})
		if ev.FallbackReason != "" {
			_ = s.recorder.Record(ctx, analytics.EventPersonalizationFallback, analytics.SourceSystem, nil, nil, nil, nil, map[string]any{
				"feature":         string(feature),
				"mode":            string(mode),
				"fallback_reason": ev.FallbackReason,
			})
		}
	}

	return personalization.Result{Text: res.Text, Provider: personalization.ProviderKimi}, ev, nil
}

// SaveInAppNotification stores an AI insight for the user.
func (s *Service) SaveInAppNotification(ctx context.Context, userID int64, feature Feature, title, body string, actions []NotificationAction) error {
	n := &Notification{
		UserID:    userID,
		Feature:   feature,
		Title:     title,
		Body:      body,
		Actions:   actions,
		CreatedAt: s.clock(),
	}
	return s.repo.SaveNotification(ctx, n)
}

// GetActiveNotifications returns non-dismissed notifications for a user.
func (s *Service) GetActiveNotifications(ctx context.Context, userID int64) ([]*Notification, error) {
	return s.repo.GetNotifications(ctx, userID, true)
}

// DismissNotification marks a notification as dismissed.
func (s *Service) DismissNotification(ctx context.Context, userID int64, id string) error {
	return s.repo.DismissNotification(ctx, userID, id)
}

// GetProofDrafts returns active proof drafts for a user.
func (s *Service) GetProofDrafts(ctx context.Context, userID int64) ([]*ProofDraft, error) {
	return s.repo.GetProofDrafts(ctx, userID, true)
}

// AcceptProofDraft returns a draft and marks it consumed.
func (s *Service) AcceptProofDraft(ctx context.Context, userID int64, id string) (*ProofDraft, error) {
	draft, err := s.repo.GetProofDraftByID(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("accept draft: %w", err)
	}
	if err := s.repo.ConsumeProofDraft(ctx, userID, id); err != nil {
		return nil, fmt.Errorf("accept draft: consume: %w", err)
	}
	return draft, nil
}

// RejectProofDraft marks a draft as consumed without using it.
func (s *Service) RejectProofDraft(ctx context.Context, userID int64, id string) error {
	return s.repo.ConsumeProofDraft(ctx, userID, id)
}
