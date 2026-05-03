package cleaner

import (
	"strings"

	"kigrepair/internal/detector"
)

func FormatDryRunSummary(report detector.DetectionReport, plan CleanupPlan, reportDir string) string {
	var b strings.Builder
	b.WriteString("Kigrepair Cleanup Dry Run\n\n")
	b.WriteString("Health before cleanup: " + string(report.Health) + "\n")
	b.WriteString("Install mode: " + string(report.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(report.InstallRoot) + "\n\n")

	if len(plan.Actions) == 0 {
		b.WriteString("No cleanup actions were required.\n\n")
	} else {
		b.WriteString("Planned actions:\n")
		for _, action := range plan.Actions {
			b.WriteString("- Would " + actionPhrase(action.Type) + ": " + action.Target + "\n")
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
	b.WriteString(reportDir)
	b.WriteString("\n")
	return b.String()
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
