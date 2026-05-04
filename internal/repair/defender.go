package repair

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/defender"
	"kigrepair/internal/detector"
)

func (w RepairWorkflow) runDefenderEnsure(ctx *app.AppContext, detect func() detector.DetectionReport, shouldRun bool) (defender.DefenderEnsureResult, bool) {
	result := defender.DefenderEnsureResult{
		StartedAt: time.Now(),
		Ensure:    true,
		ReportDir: ctx.OutputDir,
	}
	if !shouldRun {
		addOperation(ctx, "defender_ensure", "defender", app.OperationStatusSkipped, "Defender ensure was not required", "")
		result.FinishedAt = time.Now()
		result.ExitCode = 0
		return result, false
	}

	report := detect()
	result.InstallMode = string(report.InstallMode)
	result.InstallRoot = report.InstallRoot
	result.ExistingExclusions = append([]string{}, report.Defender.ExclusionPaths...)
	required, warnings := defender.RequiredPaths(report, false)
	result.RequiredPaths = required
	result.Warnings = append(result.Warnings, warnings...)
	_, result.CoveredPaths, result.MissingBefore = detector.EvaluateDefenderCoverage(required, report.Defender.ExclusionPaths)
	ctx.Logger.Info("defender required paths: %v", result.RequiredPaths)
	ctx.Logger.Info("defender missing paths before: %v", result.MissingBefore)

	if len(required) == 0 {
		addOperation(ctx, "defender_ensure", "defender", app.OperationStatusWarning, "No install root detected for Defender exclusion", "")
		result.FinishedAt = time.Now()
		result.ExitCode = defender.ExitNoInstallationRoot
		return result, true
	}
	if !report.Defender.Available {
		message := "Defender cmdlets unavailable"
		result.Warnings = append(result.Warnings, message)
		if report.Defender.Error != "" {
			result.Errors = append(result.Errors, report.Defender.Error)
		}
		addOperation(ctx, "defender_ensure", "defender", app.OperationStatusWarning, message, report.Defender.Error)
		result.FinishedAt = time.Now()
		result.ExitCode = defender.ExitAddFailed
		return result, true
	}
	if len(result.MissingBefore) == 0 {
		addOperation(ctx, "defender_ensure", "defender", app.OperationStatusSuccess, "All required Defender exclusions are already present", "")
		result.FinishedAt = time.Now()
		result.ExitCode = defender.ExitOK
		return result, true
	}

	adder := w.DefenderAdder
	if adder == nil {
		adder = defender.PowerShellExclusionAdder{}
	}
	for _, path := range result.MissingBefore {
		ctx.Logger.Info("attempting Defender exclusion add: %s", path)
		ctx.Logger.Info("executing command: powershell.exe -NoProfile -ExecutionPolicy Bypass -Command Add-MpPreference -ExclusionPath <path>")
		commandResult := adder.AddExclusion(path)
		ctx.Logger.Info("Add-MpPreference exit code for %s: %d", path, commandResult.ExitCode)
		if commandResult.ExitCode == 0 && commandResult.Err == nil {
			result.AddedPaths = append(result.AddedPaths, path)
			addOperation(ctx, "defender_ensure", path, app.OperationStatusSuccess, "Defender exclusion added", "")
			continue
		}
		errText := defenderCommandError(commandResult)
		result.FailedPaths = append(result.FailedPaths, path)
		result.Errors = append(result.Errors, errText)
		addOperation(ctx, "defender_ensure", path, app.OperationStatusFailed, "Defender exclusion add failed", errText)
	}

	final := detect()
	result.MissingAfter = detector.MissingPaths(required, final.Defender.ExclusionPaths)
	if len(result.FailedPaths) > 0 {
		result.ExitCode = defender.ExitAddFailed
	} else if len(result.MissingAfter) > 0 {
		result.ExitCode = defender.ExitVerificationFailed
		result.Errors = append(result.Errors, "Defender exclusion verification failed")
		addOperation(ctx, "defender_ensure", "defender-exclusions", app.OperationStatusFailed, "Defender exclusion verification failed", strings.Join(result.MissingAfter, "; "))
	} else {
		result.ExitCode = defender.ExitOK
		addOperation(ctx, "defender_ensure", "defender-exclusions", app.OperationStatusSuccess, "All required Defender exclusions are covered", "")
	}
	result.FinishedAt = time.Now()
	return result, true
}

func defenderCommandError(result defender.CommandResult) string {
	if strings.TrimSpace(result.Output) != "" {
		return strings.TrimSpace(result.Output)
	}
	if result.Err != nil {
		return result.Err.Error()
	}
	if result.ExitCode != 0 {
		return fmt.Sprintf("exit code %d", result.ExitCode)
	}
	return ""
}
