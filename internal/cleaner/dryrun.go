package cleaner

import (
	"strconv"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/reports"
)

func FormatDryRunSummary(report detector.DetectionReport, plan CleanupPlan, ctx *app.AppContext) string {
	var b strings.Builder
	b.WriteString("Kigrepair Cleanup Dry Run\n\n")
	b.WriteString("Health before cleanup: " + string(report.Health) + "\n")
	b.WriteString("Install mode: " + string(report.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(report.InstallRoot) + "\n\n")
	b.WriteString(processPlanSummary(plan))
	b.WriteString("\n")

	if len(plan.Actions) == 0 {
		b.WriteString("No cleanup actions were required.\n\n")
	} else {
		b.WriteString("Planned actions:\n")
		for _, action := range plan.Actions {
			b.WriteString("- Would " + actionPhrase(action.Type) + ": " + action.Target + actionDetails(action) + "\n")
		}
		b.WriteString("\n")
	}

	if len(plan.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range plan.Warnings {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}

	if len(plan.Blockers) > 0 {
		b.WriteString("Blockers:\n")
		for _, blocker := range plan.Blockers {
			b.WriteString("- " + blocker + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("No changes were made.\n\n")
	b.WriteString("Report:\n")
	b.WriteString(ctx.OutputDir)
	b.WriteString("\n")
	return reports.FormatSummary(reports.SummaryData{
		Command:       "cleanup --dry-run",
		Started:       ctx.StartedAt,
		Finished:      time.Now(),
		Mode:          string(ctx.Mode),
		ExitCode:      ctx.ExitCode,
		ReportDir:     ctx.OutputDir,
		InitialHealth: string(report.Health),
		InstallMode:   string(report.InstallMode),
		InstallRoot:   report.InstallRoot,
		Warnings:      plan.Warnings,
		Errors:        plan.Blockers,
		Actions:       dryRunActions(plan),
	}, b.String())
}

func processPlanSummary(plan CleanupPlan) string {
	terminate := 0
	skipped := 0
	for _, action := range plan.Actions {
		if action.Type == CleanupActionKillProcess {
			terminate++
		}
	}
	for _, warning := range plan.Warnings {
		if strings.Contains(strings.ToLower(warning), "skipped unsafe process") {
			skipped++
		}
	}
	return "Process cleanup plan: " + pluralCount(terminate, "trusted process", "trusted processes") + " would be terminated. " + pluralCount(skipped, "unsafe name match", "unsafe name matches") + " skipped.\n"
}

func pluralCount(count int, one string, many string) string {
	word := many
	if count == 1 {
		word = one
	}
	return strconv.Itoa(count) + " " + word
}

func dryRunActions(plan CleanupPlan) []string {
	if len(plan.Actions) == 0 {
		return []string{"No cleanup actions were required"}
	}
	actions := make([]string, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		actions = append(actions, "Would "+actionPhrase(action.Type)+": "+action.Target+actionDetails(action))
	}
	return actions
}

func actionDetails(action CleanupAction) string {
	if action.ServiceTrust == "" && action.ExecutablePath == "" {
		return ""
	}
	details := make([]string, 0, 2)
	if action.ServiceTrust != "" {
		details = append(details, "trust="+action.ServiceTrust)
	}
	if action.ExecutablePath != "" {
		details = append(details, "exe="+action.ExecutablePath)
	}
	return " (" + strings.Join(details, ", ") + ")"
}

func actionPhrase(actionType CleanupActionType) string {
	switch actionType {
	case CleanupActionMSIUninstall:
		return "run MSI uninstall"
	case CleanupActionStopService:
		return "stop service"
	case CleanupActionDeleteService:
		return "delete service"
	case CleanupActionKillProcess:
		return "kill process"
	case CleanupActionDeletePath:
		return "delete path"
	case CleanupActionDeleteRegistryKey:
		return "delete registry key"
	default:
		return "run action"
	}
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
