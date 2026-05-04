package diagnostics

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/reports"
)

func FormatCollectSummary(result CollectReportResult) string {
	var b strings.Builder
	b.WriteString("Kigrepair Collect Report\n\n")
	b.WriteString("Health: " + valueOrDash(result.Health) + "\n")
	b.WriteString("Install mode: " + valueOrDash(result.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(result.InstallRoot) + "\n\n")
	b.WriteString("Collectors:\n")
	for _, collector := range result.Collectors {
		b.WriteString("- " + collector.Name + ": " + collector.Status + "\n")
	}
	b.WriteString("\n")
	if result.BundlePath != "" {
		b.WriteString("Support bundle:\n")
		b.WriteString(result.BundlePath)
		b.WriteString("\n\n")
	}
	if len(result.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range uniqueStrings(result.Warnings) {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}
	if len(result.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, errText := range uniqueStrings(result.Errors) {
			b.WriteString("- " + errText + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Report:\n")
	b.WriteString(result.ReportDir)
	b.WriteString("\n")
	return reports.FormatSummary(reports.SummaryData{
		Command:       "collect-report",
		Started:       result.StartedAt,
		Finished:      finishedOrNow(result.FinishedAt),
		Mode:          result.Mode,
		ExitCode:      result.ExitCode,
		ReportDir:     result.ReportDir,
		InitialHealth: result.Health,
		InstallMode:   result.InstallMode,
		InstallRoot:   result.InstallRoot,
		Warnings:      result.Warnings,
		Errors:        result.Errors,
		Actions:       collectSummaryActions(result),
	}, b.String())
}

func collectSummaryActions(result CollectReportResult) []string {
	actions := make([]string, 0, len(result.Collectors)+1)
	for _, collector := range result.Collectors {
		actions = append(actions, collector.Name+": "+collector.Status)
	}
	if result.BundlePath != "" {
		actions = append(actions, fmt.Sprintf("Support bundle: %s", result.BundlePath))
	}
	return actions
}

func finishedOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
