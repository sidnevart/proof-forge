package personalization

import "errors"

// Feature identifies the personalization use-case.
type Feature string

const (
	FeatureEveningPing Feature = "evening_ping"
	FeatureAssembleProof Feature = "assemble_proof"
	FeatureLeadBriefing Feature = "lead_briefing"
)

// Mode is the team-wide privacy posture.
type Mode string

const (
	ModeOff          Mode = "off"
	ModeMetadataOnly Mode = "metadata-only"
	ModeFull         Mode = "full"
)

// Provider identifies the source of personalization.
type Provider string

const (
	ProviderTemplate Provider = "template"
	ProviderKimi     Provider = "kimi"
)

// CircuitState is the breaker state.
type CircuitState string

const (
	CircuitClosed CircuitState = "closed"
	CircuitOpen   CircuitState = "open"
)

// Result carries the validated output and metadata.
type Result struct {
	Text            string
	Provider        Provider
	PromptVersion   string
	ValidationError string
}

// InvocationEvent is the metadata written to analytics.
type InvocationEvent struct {
	Feature         Feature
	Provider        Provider
	Mode            Mode
	PromptVersion   string
	LatencyMs       int64
	TokensIn        int64
	TokensOut       int64
	FallbackReason  string
	ValidationError string
}

var (
	ErrFeatureDisabled   = errors.New("personalization feature is disabled")
	ErrBudgetExceeded    = errors.New("daily AI budget exceeded")
	ErrCircuitOpen       = errors.New("circuit breaker is open for this feature")
	ErrValidationFailed  = errors.New("LLM output failed validation")
	ErrTimeout           = errors.New("LLM provider timeout")
	ErrInvalidMode       = errors.New("invalid AI mode")
)
