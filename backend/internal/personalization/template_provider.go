package personalization

import "context"

// TemplateProvider returns deterministic fallback text for every feature.
type TemplateProvider struct{}

func NewTemplateProvider() *TemplateProvider {
	return &TemplateProvider{}
}

func (t *TemplateProvider) EveningPing(ctx context.Context, streak int, weekLogCount int) (string, error) {
	return "Что нового узнал сегодня?", nil
}

func (t *TemplateProvider) AssembleProof(ctx context.Context) (string, error) {
	return `{"candidates":[]}`, nil
}

func (t *TemplateProvider) LeadBriefing(ctx context.Context) (string, error) {
	return "Брифинг недоступен в режиме шаблонов.", nil
}
