package cleaner

import (
	"errors"
	"fmt"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
)

type CleanupWorkflow struct {
	DryRun bool
}

func (w CleanupWorkflow) Name() string {
	return "cleanup"
}

func (w CleanupWorkflow) Run(ctx *app.AppContext) error {
	if !w.DryRun {
		return errors.New("Real cleanup is not implemented yet. Run with --dry-run to preview cleanup actions.")
	}

	ctx.Logger.Info("started cleanup dry-run")
	report := detector.Detect()
	ctx.Logger.Info("detection completed")

	detectStatus := app.OperationStatusSuccess
	if report.Health != detector.GrabberHealthHealthy {
		detectStatus = app.OperationStatusWarning
	}
	detectResult := app.OperationResult{
		Step:      "cleanup.detect",
		Target:    "grabber",
		Status:    detectStatus,
		Message:   "Detection completed with health: " + string(report.Health),
		Timestamp: time.Now(),
	}
	ctx.AddResult(detectResult)
	logging.LogOperation(ctx.Logger, detectResult)

	plan := BuildPlan(report, PlanOptions{DryRun: true})
	ctx.JSONValue = plan
	ctx.Logger.Info("cleanup path validation completed")
	ctx.Logger.Info("cleanup plan generated")

	for _, action := range plan.Actions {
		result := app.OperationResult{
			Step:      "cleanup.plan." + string(action.Type),
			Target:    action.Target,
			Status:    app.OperationStatusSkipped,
			Message:   "Dry-run only: " + action.Reason,
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
	}
	for _, warning := range plan.Warnings {
		result := app.OperationResult{
			Step:      "cleanup.plan.warning",
			Target:    "cleanup-plan",
			Status:    app.OperationStatusWarning,
			Message:   warning,
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
	}
	for _, blocker := range plan.Blockers {
		result := app.OperationResult{
			Step:      "cleanup.plan.blocker",
			Target:    "cleanup-plan",
			Status:    app.OperationStatusFailed,
			Message:   blocker,
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
	}
	if len(plan.Actions) == 0 {
		result := app.OperationResult{
			Step:      "cleanup.plan",
			Target:    "cleanup-targets",
			Status:    app.OperationStatusSkipped,
			Message:   "No cleanup actions were required.",
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
	}

	if err := ctx.Reporter.WriteJSON("initial-detection", report); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteJSON("cleanup-plan", plan); err != nil {
		return err
	}
	summary := FormatDryRunSummary(report, plan, ctx.OutputDir)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}

	ctx.Logger.Info("dry-run completed without changes")
	ctx.ExitCode = cleanupDryRunExitCode(plan)
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func cleanupDryRunExitCode(plan CleanupPlan) int {
	switch {
	case len(plan.Blockers) > 0:
		return 2
	case len(plan.Warnings) > 0:
		return 1
	default:
		return 0
	}
}
