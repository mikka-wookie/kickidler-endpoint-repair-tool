package reports

import (
	"fmt"
	"strings"
	"time"
)

func FormatReportList(result ReportListResult) string {
	var b strings.Builder
	b.WriteString("Reports root: " + result.ReportsRoot + "\n\n")
	b.WriteString(fmt.Sprintf("Found reports: %d\n", result.Count))
	b.WriteString("Total size: " + FormatBytes(result.TotalSizeBytes) + "\n\n")
	for _, report := range result.Reports {
		workflow := valueOr(report.Workflow, "-")
		status := valueOr(report.Status, "-")
		bundle := "no"
		if report.HasSupportBundle {
			bundle = "yes"
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s  bundle=%s\n", report.Name, workflow, status, FormatBytes(report.SizeBytes), bundle))
	}
	if len(result.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range result.Warnings {
			b.WriteString("- " + warning + "\n")
		}
	}
	return b.String()
}

func FormatReportCleanupSummary(plan ReportCleanupPlan, result ReportCleanupResult, started time.Time) string {
	var b strings.Builder
	title := "Report cleanup"
	if result.DryRun {
		title = "Report cleanup dry-run"
	}
	b.WriteString("Kigrepair " + title + "\n\n")
	b.WriteString("Status: " + result.Status + "\n")
	b.WriteString("Reports root: " + result.ReportsRoot + "\n")
	b.WriteString(fmt.Sprintf("Policy: older than %s, keep last %d\n\n", plan.OlderThan, plan.KeepLast))
	b.WriteString(fmt.Sprintf("Candidates: %d\n", len(plan.Candidates)))
	b.WriteString(fmt.Sprintf("Planned deletes: %d\n", len(plan.PlannedDeletes)))
	b.WriteString("Estimated free space: " + FormatBytes(plan.PlannedFreeBytes) + "\n")
	if result.DryRun {
		b.WriteString("System changes made: no\n\n")
	} else {
		b.WriteString(fmt.Sprintf("Deleted report directories: %d\n", countDeleteStatus(result.Deleted, "success")))
		b.WriteString("Freed space: " + FormatBytes(result.FreedBytes) + "\n\n")
	}
	if len(plan.PlannedDeletes) > 0 {
		b.WriteString("Plan:\n")
		for _, target := range plan.PlannedDeletes {
			b.WriteString(target.Path + "\n")
		}
		b.WriteString("\n")
	}
	if len(result.Warnings) > 0 || len(plan.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range uniqueReportStrings(append(result.Warnings, plan.Warnings...)) {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}
	if len(result.Errors) > 0 || len(plan.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, errText := range uniqueReportStrings(append(result.Errors, plan.Errors...)) {
			b.WriteString("- " + errText + "\n")
		}
		b.WriteString("\n")
	}
	if result.DryRun {
		b.WriteString(fmt.Sprintf("Run to execute:\n.\\kigrepair.exe reports cleanup --older-than %s --keep-last %d --yes\n\n", plan.OlderThan, plan.KeepLast))
	}
	b.WriteString("Report directory: " + result.ReportDir + "\n")

	return FormatSummary(SummaryData{
		Command:   "reports cleanup",
		Started:   started,
		Finished:  time.Now(),
		Mode:      "cli",
		ExitCode:  result.ExitCode,
		ReportDir: result.ReportDir,
		Warnings:  uniqueReportStrings(append(result.Warnings, plan.Warnings...)),
		Errors:    uniqueReportStrings(append(result.Errors, plan.Errors...)),
		Actions: []string{
			fmt.Sprintf("Planned deletes: %d", len(plan.PlannedDeletes)),
			fmt.Sprintf("Freed bytes: %d", result.FreedBytes),
		},
	}, b.String())
}

func FormatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(div), "KMGTPE"[exp])
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func countDeleteStatus(outcomes []ReportDeleteOutcome, status string) int {
	count := 0
	for _, outcome := range outcomes {
		if outcome.Status == status {
			count++
		}
	}
	return count
}

func uniqueReportStrings(values []string) []string {
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
