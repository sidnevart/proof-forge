package companion

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateEveningPing enforces evening ping rules.
func ValidateEveningPing(text string) error {
	words := countWords(text)
	if words < 8 || words > 25 {
		return fmt.Errorf("evening ping: expected 8-25 words, got %d", words)
	}
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 || trimmed[len(trimmed)-1] != '?' {
		return fmt.Errorf("evening ping: must end with question mark")
	}
	lower := strings.ToLower(text)
	for _, bad := range []string{"молодец", "отлично", "плохо", "ты должен"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("evening ping: contains forbidden word %q", bad)
		}
	}
	return nil
}

// ValidateWeeklyRecap enforces weekly recap rules.
func ValidateWeeklyRecap(text string) error {
	words := countWords(text)
	if words < 30 || words > 150 {
		return fmt.Errorf("weekly recap: expected 30-150 words, got %d", words)
	}
	lower := strings.ToLower(text)
	for _, bad := range []string{"слабый", "сильный", "плохой", "хороший результат"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("weekly recap: contains forbidden evaluation %q", bad)
		}
	}
	return nil
}

// ValidateProofDraft enforces proof draft rules.
func ValidateProofDraft(text string) error {
	words := countWords(text)
	if words < 20 || words > 80 {
		return fmt.Errorf("proof draft: expected 20-80 words, got %d", words)
	}
	lower := strings.ToLower(text)
	for _, bad := range []string{"слабый", "плохой", "хороший", "отлично", "молодец"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("proof draft: contains forbidden evaluation %q", bad)
		}
	}
	return nil
}

// ValidateBuddyStalled enforces buddy stalled alert rules.
func ValidateBuddyStalled(text string) error {
	words := countWords(text)
	if words < 15 || words > 40 {
		return fmt.Errorf("buddy stalled: expected 15-40 words, got %d", words)
	}
	lower := strings.ToLower(text)
	for _, bad := range []string{"слабый", "плохой", "лень", "тормозит"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("buddy stalled: contains forbidden word %q", bad)
		}
	}
	return nil
}

// ValidateLeadBrief enforces lead brief rules.
func ValidateLeadBrief(text string) error {
	words := countWords(text)
	if words < 30 || words > 80 {
		return fmt.Errorf("lead brief: expected 30-80 words, got %d", words)
	}
	lower := strings.ToLower(text)
	for _, bad := range []string{"слабый работник", "плохой работник", "сильный работник", "слабый", "плохой результат"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("lead brief: contains forbidden evaluation %q", bad)
		}
	}
	// Must contain at least one risk section or explicit "Без рисков".
	if !strings.Contains(lower, "без рисков") && !strings.Contains(lower, "риск") {
		return fmt.Errorf("lead brief: must contain risk section or explicit 'Без рисков'")
	}
	return nil
}

// countWords counts words in a string (simple split on whitespace/punctuation).
func countWords(s string) int {
	var count int
	inWord := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !inWord {
				count++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return count
}
