package recaps

import (
	"fmt"
	"strings"
	"time"
)

// BuildPrompt constructs the AI summarization prompt.
// The AI is instructed to produce observational text only — it must NOT approve,
// reject, or evaluate the quality or sufficiency of proof evidence.
func BuildPrompt(goal GoalForRecap, period [2]time.Time, checkIns []ApprovedCheckIn) string {
	var b strings.Builder

	b.WriteString("You are a progress reporter for an accountability app called ProofForge.\n")
	b.WriteString("Your role is to write a factual weekly recap based on the evidence the goal owner submitted.\n")
	b.WriteString("IMPORTANT: Do NOT approve, reject, or evaluate evidence quality. ")
	b.WriteString("Do NOT make any decisions about goal progress. Only describe what happened.\n\n")

	fmt.Fprintf(&b, "Goal: %s\n", goal.Title)
	if goal.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", goal.Description)
	}
	fmt.Fprintf(&b, "Period: %s – %s\n\n",
		period[0].Format("2006-01-02"),
		period[1].Format("2006-01-02"),
	)

	if len(checkIns) == 0 {
		b.WriteString("No approved check-ins were found for this period.\n")
	} else {
		fmt.Fprintf(&b, "Approved check-ins (%d):\n", len(checkIns))
		for i, ci := range checkIns {
			fmt.Fprintf(&b, "\nCheck-in %d (approved %s):\n", i+1, ci.ApprovedAt.Format("2006-01-02"))
			for _, e := range ci.Evidence {
				switch e.Kind {
				case "text":
					fmt.Fprintf(&b, "  - Note: %s\n", e.TextContent)
				case "link":
					fmt.Fprintf(&b, "  - Link: %s\n", e.ExternalURL)
				default:
					fmt.Fprintf(&b, "  - File attachment (%s)\n", e.Kind)
				}
			}
			if len(ci.Evidence) == 0 {
				b.WriteString("  (no evidence items)\n")
			}
		}
	}

	b.WriteString("\nWrite a concise 2–4 sentence weekly summary in English. ")
	b.WriteString("Describe what was accomplished based on the evidence. ")
	b.WriteString("Do not include any judgment about whether the evidence is sufficient or whether the goal was achieved.")
	return b.String()
}
