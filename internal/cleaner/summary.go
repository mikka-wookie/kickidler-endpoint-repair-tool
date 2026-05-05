package cleaner

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/reports"
	"kigrepair/internal/rollback"
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

func FormatRealCleanupSummary(initial detector.DetectionReport, final *detector.DetectionReport, plan CleanupPlan, results []app.OperationResult, ctx *app.AppContext, rollbackInfo *rollback.RollbackInfo) string {
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
	b.WriteString(fmt.Sprintf("Exit code: %d\n\n", ctx.ExitCode))
	b.WriteString("Report:\n")
	b.WriteString(ctx.OutputDir)
	b.WriteString("\n")
	if rollbackInfo != nil {
		b.WriteString("\n")
		b.WriteString(rollback.FormatSummarySection(rollbackInfo))
	}
	finalHealth := ""
	installMode := string(initial.InstallMode)
	installRoot := initial.InstallRoot
	primaryService := initial.PrimaryService
	if final != nil {
		finalHealth = string(final.Health)
		installMode = string(final.InstallMode)
		installRoot = final.InstallRoot
		primaryService = final.PrimaryService
	}
	return reports.FormatSummary(reports.SummaryData{
		Command:        "cleanup",
		Started:        ctx.StartedAt,
		Finished:       time.Now(),
		Mode:           string(ctx.Mode),
		ExitCode:       ctx.ExitCode,
		ReportDir:      ctx.OutputDir,
		InitialHealth:  string(initial.Health),
		FinalHealth:    finalHealth,
		InstallMode:    installMode,
		InstallRoot:    installRoot,
		PrimaryService: primaryService,
		Warnings:       cleanupSummaryMessages(results, app.OperationStatusWarning),
		Errors:         cleanupSummaryMessages(results, app.OperationStatusFailed),
		Actions:        cleanupSummaryActions(results, plan),
	}, b.String())
}

func cleanupSummaryMessages(results []app.OperationResult, status app.OperationStatus) []string {
	messages := make([]string, 0)
	for _, result := range results {
		if result.Status != status {
			continue
		}
		if result.Error != "" {
			messages = append(messages, result.Error)
		} else {
			messages = append(messages, result.Message)
		}
	}
	return messages
}

func cleanupSummaryActions(results []app.OperationResult, plan CleanupPlan) []string {
	actions := make([]string, 0)
	for _, result := range results {
		if isExecutionResult(result) {
			actions = append(actions, fmt.Sprintf("%s %s: %s", result.Step, result.Target, result.Status))
		}
	}
	if len(actions) == 0 && len(plan.Actions) == 0 {
		return []string{"No cleanup actions were required"}
	}
	if len(actions) == 0 {
		return []string{"No cleanup actions executed"}
	}
	return actions
}

func writePlannedActions(b *strings.Builder, plan CleanupPlan) {
	if len(plan.Actions) == 0 {
		b.WriteString("Planned actions:\n- No cleanup actions were required.\n\n")
		return
	}
	b.WriteString(processPlanSummary(plan))
	b.WriteString("\n")
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
