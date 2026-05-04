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
	final := detect()
	ctx.Logger.Info("verification detection health/mode/root: %s / %s / %s", final.Health, final.InstallMode, final.InstallRoot)
	if err := ctx.Reporter.WriteJSON("final-detection", final); err != nil {
		return err
	}

	result := Verify(final, Options{InstallExecuted: w.InstallExecuted, MSIInstallLog: w.MSIInstallLog})
	result.Command = "verify"
	result.StartedAt = startedAt
	result.FinishedAt = time.Now()
	result.Mode = string(ctx.Mode)
	result.ReportDir = ctx.OutputDir
	result.ExitCode = ExitCode(result.Status)
	ctx.ExitCode = result.ExitCode
	ctx.JSONValue = result

	operation := app.OperationResult{
		Step:      "verify.final",
		Target:    "grabber",
		Status:    operationStatus(result.Status),
		Message:   result.Message,
		Error:     strings.Join(result.Errors, "; "),
		Timestamp: time.Now(),
	}
	ctx.AddResult(operation)
	logging.LogOperation(ctx.Logger, operation)

	if err := ctx.Reporter.WriteJSON("verification-result", result); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	summary := FormatSummary(result)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func ExitCode(status VerificationStatus) int {
	switch status {
	case VerificationSuccess:
		return app.ExitSuccess
	case VerificationWarning:
		return app.ExitWarnings
	default:
		return app.ExitVerificationFailed
	}
}

func operationStatus(status VerificationStatus) app.OperationStatus {
	switch status {
	case VerificationSuccess:
		return app.OperationStatusSuccess
	case VerificationWarning:
		return app.OperationStatusWarning
	default:
		return app.OperationStatusFailed
	}
}
