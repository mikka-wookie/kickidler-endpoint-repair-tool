package recommendations

import (
	"strings"
)

func SupportSummary(result RecommendationResult) string {
	if result.PrimaryAction == nil {
		return "No primary recommendation could be selected."
	}
	action := result.PrimaryAction
	var b strings.Builder
	b.WriteString(action.Code)
	if action.Command != "" {
		b.WriteString(": ")
		b.WriteString(action.Command)
	}
	if action.Reason != "" {
		b.WriteString(" - ")
		b.WriteString(action.Reason)
	}
	return b.String()
}

func FormatSection(result RecommendationResult) string {
	var b strings.Builder
	b.WriteString("Recommendation\n")
	b.WriteString("--------------\n")
	b.WriteString("Status: " + valueOrDash(result.Status) + "\n")
	if result.PrimaryAction == nil {
		b.WriteString("Primary action: -\n")
		return b.String()
	}
	action := result.PrimaryAction
	b.WriteString("Primary action: " + action.Code + "\n")
	b.WriteString("Requires admin: " + yesNo(action.RequiresAdmin) + "\n")
	b.WriteString("Destructive: " + yesNo(action.Destructive) + "\n")
	if len(action.RequiredInputs) > 0 {
		b.WriteString("Required inputs: " + strings.Join(action.RequiredInputs, ", ") + "\n")
	}
	if action.Command != "" {
		b.WriteString("Command:\n")
		b.WriteString(action.Command + "\n")
	}
	b.WriteString("\nReason:\n")
	b.WriteString(valueOrDash(action.Reason) + "\n")
	if len(result.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range result.Warnings {
			b.WriteString("- " + warning + "\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func FormatConsole(result RecommendationResult) string {
	if result.PrimaryAction == nil {
		return "Recommended action: -\n"
	}
	action := result.PrimaryAction
	var b strings.Builder
	b.WriteString("Recommended action: " + action.Code + "\n")
	if action.Command != "" {
		b.WriteString("Command: " + action.Command + "\n")
	}
	b.WriteString("Requires admin: " + yesNo(action.RequiresAdmin) + "\n")
	if len(action.RequiredInputs) > 0 {
		b.WriteString("Required inputs: " + strings.Join(action.RequiredInputs, ", ") + "\n")
	}
	return b.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
