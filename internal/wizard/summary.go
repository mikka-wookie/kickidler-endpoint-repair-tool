package wizard

import (
	"fmt"
	"strings"
)

func FormatSummary(result Result) string {
	var b strings.Builder
	b.WriteString("Wizard\n")
	b.WriteString("------\n")
	b.WriteString("Status: " + result.Status + "\n")
	b.WriteString("Repair executed: " + yesNo(result.RepairExecuted) + "\n")
	b.WriteString("Report directory: " + result.ReportDir + "\n\n")
	b.WriteString("Steps:\n")
	for _, step := range result.Steps {
		b.WriteString(fmt.Sprintf("- %s: %s", step.Name, step.Status))
		if step.Summary != "" {
			b.WriteString(" (" + step.Summary + ")")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if result.ClassificationCode != "" {
		b.WriteString("Primary issue: " + result.ClassificationCode + "\n")
	}
	if result.RecommendationCode != "" {
		b.WriteString("Recommended action: " + result.RecommendationCode + "\n")
	}
	if result.PreflightStatus != "" {
		b.WriteString("Preflight: " + result.PreflightStatus + "\n")
	}
	if result.RepairPlanStatus != "" {
		b.WriteString("Repair readiness: " + result.RepairPlanStatus + "\n")
	}
	if result.BundlePath != "" {
		b.WriteString("Support bundle: " + result.BundlePath + "\n")
	}
	b.WriteString("\nNext command:\n")
	b.WriteString(`.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes`)
	b.WriteString("\n")
	return b.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
