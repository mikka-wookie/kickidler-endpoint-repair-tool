package preflight

import (
	"strings"
	"time"

	"kigrepair/internal/reports"
)

func FormatSummary(result Result) string {
	var b strings.Builder
	b.WriteString("Preflight\n")
	b.WriteString("---------\n")
	b.WriteString("Status: " + result.Status + "\n")
	b.WriteString("Ready for repair: " + yesNo(result.ReadyForRepair) + "\n")
	b.WriteString("Report directory: " + valueOrDash(result.ReportDir) + "\n")
	if result.Installer.Path != "" {
		b.WriteString("Installer: " + result.Installer.Path + "\n")
	}
	if result.Installer.SHA256 != "" {
		b.WriteString("Installer SHA-256: " + result.Installer.SHA256 + "\n")
	}
	b.WriteString("\nRequired checks:\n")
	for _, check := range result.Checks {
		if !check.Required {
			continue
		}
		writeCheckLine(&b, check)
	}
	writeCheckGroup(&b, "\nWarnings", result.Checks, CheckWarning)
	b.WriteString("\nClassification:\n")
	if result.Classification.PrimaryIssue != nil {
		b.WriteString("Primary issue: " + result.Classification.PrimaryIssue.Code + "\n")
	} else {
		b.WriteString("Primary issue: none\n")
	}
	b.WriteString("\nRecommendation:\n")
	if result.Recommendation.PrimaryAction != nil {
		b.WriteString("Primary action: " + result.Recommendation.PrimaryAction.Code + "\n")
		if result.Recommendation.PrimaryAction.Command != "" {
			b.WriteString("Command:\n")
			b.WriteString(result.Recommendation.PrimaryAction.Command + "\n")
		}
	} else {
		b.WriteString("Primary action: none\n")
	}

	return reports.FormatSummary(reports.SummaryData{
		Command:     "preflight",
		Started:     parseTime(result.CreatedAt),
		Finished:    time.Now(),
		ExitCode:    result.ExitCode,
		ReportDir:   result.ReportDir,
		Warnings:    result.Warnings,
		Errors:      result.Errors,
		Actions:     []string{"Preflight: " + result.Status},
		InstallMode: result.InstallMode,
		InstallRoot: result.InstallRoot,
	}, b.String())
}

func FormatConsoleSummary(result Result) string {
	var b strings.Builder
	b.WriteString("Preflight: " + result.Status + "\n")
	b.WriteString("Ready for repair: " + yesNo(result.ReadyForRepair) + "\n")
	b.WriteString("Report directory: " + valueOrDash(result.ReportDir) + "\n\n")
	failed := checksByStatus(result.Checks, CheckFail, true)
	if len(failed) > 0 {
		b.WriteString("Failed required checks:\n")
		for _, check := range failed {
			b.WriteString("- " + check.Code + ": " + firstNonEmpty(check.Action, check.Description, "Fix this required check.") + "\n")
		}
		b.WriteString("\n")
	}
	warnings := checksByStatus(result.Checks, CheckWarning, false)
	if len(warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, check := range warnings {
			b.WriteString("- " + check.Code + ": " + firstNonEmpty(check.Description, check.Action, "Review this warning.") + "\n")
		}
		b.WriteString("\n")
	}
	if result.ReadyForRepair {
		b.WriteString("Recommended command:\n")
	} else {
		b.WriteString("Recommended action:\n")
		b.WriteString("Fix failed preflight checks, then run:\n")
	}
	if result.Recommendation.PrimaryAction != nil && result.Recommendation.PrimaryAction.Command != "" {
		b.WriteString(result.Recommendation.PrimaryAction.Command + "\n")
	}
	return b.String()
}

func writeCheckLine(b *strings.Builder, check CheckResult) {
	b.WriteString("- " + check.Code + ": " + check.Status)
	if check.Action != "" && check.Status != CheckPass {
		b.WriteString(" - " + check.Action)
	} else if check.Evidence != "" {
		b.WriteString(" - " + check.Evidence)
	}
	b.WriteString("\n")
}

func writeCheckGroup(b *strings.Builder, title string, checks []CheckResult, status string) {
	filtered := checksByStatus(checks, status, false)
	b.WriteString(title + ":\n")
	if len(filtered) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, check := range filtered {
		writeCheckLine(b, check)
	}
}

func checksByStatus(checks []CheckResult, status string, requiredOnly bool) []CheckResult {
	result := []CheckResult{}
	for _, check := range checks {
		if check.Status != status {
			continue
		}
		if requiredOnly && !check.Required {
			continue
		}
		result = append(result, check)
	}
	return result
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}
