package recaps

import "context"

// NoopProvider returns a static stub summary. Used when OPENAI_API_KEY is not set.
type NoopProvider struct{}

func (NoopProvider) Summarize(_ context.Context, _ string) (string, string, error) {
	return "AI summarization is disabled. Set OPENAI_API_KEY to enable weekly recaps.", "noop", nil
}
