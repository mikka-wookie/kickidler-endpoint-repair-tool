package cleaner

import (
	"fmt"
	"strings"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

func FormatRealCleanupPreflight(report detector.DetectionReport, plan CleanupPlan, reportDir string) string {
	var b strings.Builder
	b.WriteString("Kigrepair Cleanup\n\n")
	b.WriteString("Initial health: " + string(report.Health) + "\n")
	b.WriteString("Install mode: " + string(report.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(report.InstallRoot) + "\n\n")
	writePlannedActions(&b, plan)
	b.WriteString("Confirmation:\n")
	return b.String()
}

func FormatRealCleanupSummary(initial detector.DetectionReport, final *detector.DetectionReport, plan CleanupPlan, results []app.OperationResult, reportDir string, exitCode int) string {
	var b strings.Builder
	b.WriteString("Kigrepair Cleanup\n\n")
	b.WriteString("Initial health: " + string(initial.Health) + "\n")
	b.WriteString("Install mode: " + string(initial.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(initial.InstallRoot) + "\n\n")
	writePlannedActions(&b, plan)

	b.WriteString("Execution result:\n")
	wroteExecution := false
	for _, result := range results {
		if !isExecutionResult(result) {
			continue
		}
		wroteExecution = true
		b.WriteString(fmt.Sprintf("- %s %s: %s", result.Step, result.Target, result.Status))
		if result.Message != "" {
			b.WriteString(" (" + result.Message + ")")
		}
		b.WriteString("\n")
	}
	if !wroteExecution {
		b.WriteString("- No cleanup actions executed.\n")
	}
	b.WriteString("\n")

	if final != nil {
		b.WriteString("Final health: " + string(final.Health) + "\n")
		if len(final.Issues) > 0 {
			b.WriteString("\nFinal issues:\n")
			for _, issue := range final.Issues {
				b.WriteString("- " + issue + "\n")
			}
		}
	} else {
		b.WriteString("Final health: not checked\n")
	}
	b.WriteString(fmt.Sprintf("Exit code: %d\n\n", exitCode))
	b.WriteString("Report:\n")
	b.WriteString(reportDir)
	b.WriteString("\n")
	return b.String()
}

func writePlannedActions(b *strings.Builder, plan CleanupPlan) {
	if len(plan.Actions) == 0 {
		b.WriteString("Planned actions:\n- No cleanup actions were required.\n\n")
		return
	}
	b.WriteString("Planned actions:\n")
	for _, action := range plan.Actions {
		b.WriteString("- " + displayActionPhrase(action.Type) + ": " + action.Target + "\n")
	}
	b.WriteString("\n")
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
}

func displayActionPhrase(actionType CleanupActionType) string {
	switch actionType {
	case CleanupActionMSIUninstall:
		return "Run MSI uninstall"
	case CleanupActionStopService:
		return "Stop service"
	case CleanupActionDeleteService:
		return "Delete service"
	case CleanupActionKillProcess:
		return "Kill process"
	case CleanupActionDeletePath:
		return "Delete path"
	case CleanupActionDeleteRegistryKey:
		return "Delete registry key"
	default:
		return "Run action"
	}
}

func isExecutionResult(result app.OperationResult) bool {
	switch CleanupActionType(result.Step) {
	case CleanupActionStopService,
		CleanupActionKillProcess,
		CleanupActionMSIUninstall,
		CleanupActionDeleteService,
		CleanupActionDeletePath,
		CleanupActionDeleteRegistryKey:
		return true
	default:
		return false
	}
}
