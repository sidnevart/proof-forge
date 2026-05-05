package ai

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestFakeGoalRefineProviderReturnsThreeDeterministicVariants(t *testing.T) {
	provider := FakeGoalRefineProvider{}
	ctx := context.Background()
	draftText := "  Учить Англ для собеседований  "

	first, err := provider.RefineGoal(ctx, draftText)
	if err != nil {
		t.Fatalf("RefineGoal() first call error = %v", err)
	}

	second, err := provider.RefineGoal(ctx, draftText)
	if err != nil {
		t.Fatalf("RefineGoal() second call error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("RefineGoal() returned non-deterministic result:\nfirst: %#v\nsecond: %#v", first, second)
	}

	if first.Category != "учёба" {
		t.Fatalf("expected category %q, got %q", "учёба", first.Category)
	}

	if len(first.Variants) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(first.Variants))
	}

	for i, variant := range first.Variants {
		if variant.Title == "" {
			t.Fatalf("variant %d expected non-empty title", i)
		}
		if variant.Smart == "" {
			t.Fatalf("variant %d expected non-empty smart text", i)
		}
		if len(variant.ProofExamples) != 3 {
			t.Fatalf("variant %d expected 3 proof examples, got %d", i, len(variant.ProofExamples))
		}
	}

	if !strings.Contains(first.Variants[0].Smart, "Учить Англ для собеседований") {
		t.Fatalf("expected user-facing smart text to preserve trimmed original input, got %q", first.Variants[0].Smart)
	}
}
