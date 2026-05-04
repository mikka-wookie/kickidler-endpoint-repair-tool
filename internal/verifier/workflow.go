package verifier

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
)

type VerifyWorkflow struct {
	InstallExecuted bool
	MSIInstallLog   string

	Detect func() detector.DetectionReport
}

func (w VerifyWorkflow) Name() string {
	return "verify"
}

func (w VerifyWorkflow) Run(ctx *app.AppContext) error {
	startedAt := time.Now()
	ctx.Logger.Info("verify started")
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	ctx.Logger.Info("detection start")
	final := detect()
	ctx.Logger.Info("detection end: health/mode/root: %s / %s / %s", final.Health, final.InstallMode, final.InstallRoot)
	detectionOperation := app.OperationResult{
		Step:      "verify.detection",
		Target:    "grabber",
		Status:    detectionOperationStatus(final),
		Message:   "Detection completed with health: " + string(final.Health),
		Timestamp: time.Now(),
	}
	ctx.AddResult(detectionOperation)
	logging.LogOperation(ctx.Logger, detectionOperation)
	if err := ctx.Reporter.WriteJSON("initial-detection", final); err != nil {
		return err
	}
	ctx.Logger.Info("wrote initial-detection.json")

	ctx.Logger.Info("verification start")
	result := VerifyInstallation(final, VerifyOptions{
		InstallExecuted:            w.InstallExecuted,
		MSIInstallLogPath:          w.MSIInstallLog,
		RequireRunningProcess:      true,
		AllowDefenderUnavailable:   true,
		ExpectInstalledState:       true,
		AllowStoppedServiceWarning: true,
	})
	result.Command = "verify"
	result.StartedAt = startedAt
	result.FinishedAt = time.Now()
	result.Mode = string(ctx.Mode)
	result.ReportDir = ctx.OutputDir
	result.ExitCode = ExitCode(result.OverallStatus)
	ctx.ExitCode = result.ExitCode
	ctx.JSONValue = result
	ctx.Logger.Info("verification end: status=%s", result.OverallStatus)

	operation := app.OperationResult{
		Step:      "verify.final",
		Target:    "grabber",
		Status:    operationStatus(result.OverallStatus),
		Message:   "Final verification result: " + string(result.OverallStatus),
		Error:     strings.Join(result.Errors, "; "),
		Timestamp: time.Now(),
	}
	ctx.AddResult(operation)
	logging.LogOperation(ctx.Logger, operation)

	if err := ctx.Reporter.WriteJSON("verification-result", result); err != nil {
		return err
	}
	ctx.Logger.Info("wrote verification-result.json")
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	ctx.Logger.Info("wrote operations.json")
	summary := FormatSummary(result)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	ctx.Logger.Info("wrote summary.txt")
	ctx.Logger.Info("final status: %s", result.OverallStatus)
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatConsoleSummary(result))
	}
	return nil
}

func ExitCode(status VerificationStatus) int {
	switch status {
	case VerificationPassed:
		return app.ExitSuccess
	case VerificationWarning, VerificationSkipped:
		return app.ExitWarnings
	default:
		return app.ExitVerificationFailed
	}
}

func detectionOperationStatus(report detector.DetectionReport) app.OperationStatus {
	switch report.Health {
	case detector.GrabberHealthHealthy:
		return app.OperationStatusSuccess
	case detector.GrabberHealthUnknown:
		return app.OperationStatusFailed
	default:
		return app.OperationStatusWarning
	}
}
