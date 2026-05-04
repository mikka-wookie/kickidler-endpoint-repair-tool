package repair

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
	"kigrepair/internal/logging"
)

func addOperation(ctx *app.AppContext, step string, target string, status app.OperationStatus, message string, errText string) {
	result := app.OperationResult{
		Step:      step,
		Target:    target,
		Status:    status,
		Message:   message,
		Error:     errText,
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
}

func addPlanOperations(ctx *app.AppContext, plan cleaner.CleanupPlan) {
	status := app.OperationStatusSuccess
	message := fmt.Sprintf("Cleanup plan generated with %d action(s)", len(plan.Actions))
	if len(plan.Blockers) > 0 {
		status = app.OperationStatusFailed
		message = "Cleanup plan has blockers"
	} else if len(plan.Warnings) > 0 {
		status = app.OperationStatusWarning
	}
	addOperation(ctx, "cleanup_plan", "cleanup-targets", status, message, strings.Join(plan.Blockers, "; "))
	for _, warning := range plan.Warnings {
		addOperation(ctx, "cleanup_plan", "cleanup-warning", app.OperationStatusWarning, warning, "")
	}
	for _, blocker := range plan.Blockers {
		addOperation(ctx, "cleanup_plan", "cleanup-blocker", app.OperationStatusFailed, blocker, blocker)
	}
}

func cleanupResult(plan cleaner.CleanupPlan, results []app.OperationResult) struct {
	Plan       cleaner.CleanupPlan   `json:"plan"`
	Operations []app.OperationResult `json:"operations"`
} {
	return struct {
		Plan       cleaner.CleanupPlan   `json:"plan"`
		Operations []app.OperationResult `json:"operations"`
	}{Plan: plan, Operations: results}
}

func installResult(started time.Time, mode string, installerPath string, inviteProvided bool, msi installer.MSIResult, reportDir string) installer.InstallResult {
	return installer.InstallResult{
		StartedAt:      started,
		FinishedAt:     time.Now(),
		Mode:           mode,
		InstallerPath:  installerPath,
		InviteProvided: inviteProvided,
		MSI:            msi,
		ReportDir:      reportDir,
	}
}

func detectionStatus(report detector.DetectionReport) app.OperationStatus {
	if report.Health == detector.GrabberHealthHealthy || report.Health == detector.GrabberHealthNotInstalled {
		return app.OperationStatusSuccess
	}
	return app.OperationStatusWarning
}

func decisionStatus(decision Decision) app.OperationStatus {
	if decision.Stop || len(decision.Errors) > 0 {
		return app.OperationStatusWarning
	}
	return app.OperationStatusSuccess
}

func decisionMessage(decision Decision) string {
	if decision.Stop {
		return "Repair stopped by initial health decision"
	}
	return fmt.Sprintf("cleanup=%t install=%t defender=%t", decision.CleanupNeeded, decision.InstallNeeded, decision.DefenderNeeded)
}

func verificationStatus(verification Verification) app.OperationStatus {
	switch verification.Status {
	case "success":
		return app.OperationStatusSuccess
	case "warning":
		return app.OperationStatusWarning
	default:
		return app.OperationStatusFailed
	}
}

func cleanupFailed(results []app.OperationResult) bool {
	for _, result := range results {
		if isCleanupExecutionStep(result.Step) && result.Status == app.OperationStatusFailed {
			return true
		}
	}
	return false
}

func cleanupWarning(results []app.OperationResult) bool {
	for _, result := range results {
		if isCleanupExecutionStep(result.Step) && result.Status == app.OperationStatusWarning {
			return true
		}
	}
	return false
}

func isCleanupExecutionStep(step string) bool {
	switch cleaner.CleanupActionType(step) {
	case cleaner.CleanupActionMSIUninstall,
		cleaner.CleanupActionStopService,
		cleaner.CleanupActionDeleteService,
		cleaner.CleanupActionKillProcess,
		cleaner.CleanupActionDeletePath,
		cleaner.CleanupActionDeleteRegistryKey:
		return true
	default:
		return false
	}
}

func finalExitCode(msi installer.MSIResult, verification Verification, results []app.OperationResult, defenderRan bool) int {
	if !msi.Success {
		return ExitInstallFailed
	}
	if verification.Status == "failed" {
		return ExitVerificationFailed
	}
	if msi.RebootRequired && verification.Status != "failed" {
		return ExitRebootRequired
	}
	if verification.Status == "warning" || cleanupWarning(results) || cleanupFailed(results) || defenderFailed(results) {
		return ExitWarnings
	}
	if defenderRan && defenderFailed(results) {
		return ExitWarnings
	}
	return ExitSuccess
}

func defenderFailed(results []app.OperationResult) bool {
	for _, result := range results {
		if result.Step == "defender_ensure" && result.Status == app.OperationStatusFailed {
			return true
		}
	}
	return false
}

func classifyOperation(result *RepairResult, operation app.OperationResult) {
	if result == nil {
		return
	}
	if operation.Status == app.OperationStatusWarning {
		result.Warnings = append(result.Warnings, operation.Message)
	}
	if operation.Status == app.OperationStatusFailed {
		if operation.Error != "" {
			result.Errors = append(result.Errors, operation.Error)
		} else {
			result.Errors = append(result.Errors, operation.Message)
		}
	}
}
