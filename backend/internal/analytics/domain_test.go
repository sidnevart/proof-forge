package analytics

import (
	"testing"
)

func TestValidateProperties_AllowsSafeKeys(t *testing.T) {
	props := map[string]any{
		"feature":   "evening_ping",
		"mode":      "full",
		"latency_ms": 120,
	}
	if err := ValidateProperties(props); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateProperties_RejectsForbiddenKeys(t *testing.T) {
	for _, key := range ForbiddenKeys {
		props := map[string]any{key: "sensitive"}
		if err := ValidateProperties(props); err == nil {
			t.Fatalf("expected error for forbidden key %q, got nil", key)
		}
	}
}

func TestAllowedFrontendEvents(t *testing.T) {
	allowed := []EventName{EventTeamFeedOpened, EventNotificationClicked}
	for _, ev := range allowed {
		if !AllowedFrontendEvents[ev] {
			t.Fatalf("expected %q to be allowed", ev)
		}
	}

	notAllowed := []EventName{EventDailyLogSubmitted, EventProofApproved, EventPersonalizationInvoked}
	for _, ev := range notAllowed {
		if AllowedFrontendEvents[ev] {
			t.Fatalf("expected %q to NOT be allowed from frontend", ev)
		}
	}
}
