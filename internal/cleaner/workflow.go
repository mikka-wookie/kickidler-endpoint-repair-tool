package cleaner

import (
	"errors"
	"fmt"
	"os"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
)

type CleanupWorkflow struct {
	DryRun bool
	Yes    bool
}

func (w CleanupWorkflow) Name() string {
	return "cleanup"
}

func (w CleanupWorkflow) Run(ctx *app.AppContext) error {
	mode := "real"
	if w.DryRun {
		mode = "dry-run"
	}
	ctx.Logger.Info("cleanup mode: %s", mode)
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)

	report := detector.Detect()
	ctx.Logger.Info("admin status: %t", report.IsAdmin)
	ctx.Logger.Info("detection completed")
	ctx.Logger.Info("initial health: %s", report.Health)

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

	plan := BuildPlan(report, PlanOptions{DryRun: w.DryRun})
	plan.Actions = SortActionsForExecution(plan.Actions)
	ctx.Logger.Info("cleanup path validation completed")
	ctx.Logger.Info("cleanup plan generated")
	ctx.Logger.Info("cleanup plan action count: %d", len(plan.Actions))
	for _, blocker := range plan.Blockers {
		ctx.Logger.Error("cleanup blocker: %s", blocker)
	}
	for _, warning := range plan.Warnings {
		ctx.Logger.Warn("cleanup warning: %s", warning)
	}

	if w.DryRun {
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

	if !w.DryRun {
		ctx.JSONValue = CleanupPlanResult{
			Command:          "cleanup",
			InitialDetection: report,
			CleanupPlan:      plan,
			Operations:       ctx.Results,
			ExitCode:         ctx.ExitCode,
			ReportDir:        ctx.OutputDir,
			Warnings:         plan.Warnings,
			Errors:           plan.Blockers,
		}
		return w.runRealCleanup(ctx, report, plan)
	}

	ctx.ExitCode = cleanupDryRunExitCode(plan)
	ctx.JSONValue = CleanupPlanResult{
		Command:          "cleanup --dry-run",
		InitialDetection: report,
		CleanupPlan:      plan,
		Operations:       ctx.Results,
		ExitCode:         ctx.ExitCode,
		ReportDir:        ctx.OutputDir,
		Warnings:         plan.Warnings,
		Errors:           plan.Blockers,
	}
	summary := FormatDryRunSummary(report, plan, ctx)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}

	ctx.Logger.Info("dry-run completed without changes")
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func (w CleanupWorkflow) runRealCleanup(ctx *app.AppContext, initial detector.DetectionReport, plan CleanupPlan) error {
	if len(plan.Blockers) > 0 || hasUnsafeAction(plan) {
		ctx.ExitCode = app.ExitInvalidInput
		result := app.OperationResult{
			Step:      "cleanup.blocked",
			Target:    "cleanup-plan",
			Status:    app.OperationStatusFailed,
			Message:   "Cleanup blocked by safety validation",
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
		ctx.JSONValue = CleanupExecutionResult{
			InitialDetection: initial,
			CleanupPlan:      plan,
			Operations:       ctx.Results,
			ExitCode:         ctx.ExitCode,
			ReportDir:        ctx.OutputDir,
		}
		summary := FormatRealCleanupSummary(initial, nil, plan, ctx.Results, ctx)
		_ = ctx.Reporter.WriteText("summary", summary)
		_ = ctx.Reporter.WriteOperations(ctx.Results)
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Print(summary)
		}
		return nil
	}

	if (ctx.Quiet || ctx.NonInteractive) && !w.Yes {
		ctx.ExitCode = app.ExitConfirmationRequired
		message := "Real cleanup in non-interactive mode requires --yes"
		result := app.OperationResult{
			Step:      "cleanup.confirmation",
			Target:    "user",
			Status:    app.OperationStatusFailed,
			Message:   message,
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
		ctx.Logger.Warn("confirmation status: missing")
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Fprintln(os.Stderr, message)
		}
		ctx.JSONValue = CleanupExecutionResult{
			InitialDetection: initial,
			CleanupPlan:      plan,
			Operations:       ctx.Results,
			ExitCode:         ctx.ExitCode,
			ReportDir:        ctx.OutputDir,
		}
		_ = ctx.Reporter.WriteOperations(ctx.Results)
		_ = ctx.Reporter.WriteText("summary", FormatRealCleanupSummary(initial, nil, plan, ctx.Results, ctx))
		return nil
	}

	if !checks.IsAdmin() {
		ctx.ExitCode = app.ExitAdminRequired
		message := "Administrator rights are required for real cleanup"
		result := app.OperationResult{
			Step:      "cleanup.admin",
			Target:    "administrator",
			Status:    app.OperationStatusFailed,
			Message:   message,
			Timestamp: time.Now(),
		}
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
		ctx.Logger.Error(message)
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Fprintln(os.Stderr, message)
		}
		ctx.JSONValue = CleanupExecutionResult{
			InitialDetection: initial,
			CleanupPlan:      plan,
			Operations:       ctx.Results,
			ExitCode:         ctx.ExitCode,
			ReportDir:        ctx.OutputDir,
		}
		_ = ctx.Reporter.WriteOperations(ctx.Results)
		_ = ctx.Reporter.WriteText("summary", FormatRealCleanupSummary(initial, nil, plan, ctx.Results, ctx))
		return nil
	}

	if len(plan.Actions) > 0 {
		if !ctx.Quiet && !ctx.JSONOutput && !w.Yes {
			fmt.Print(FormatRealCleanupPreflight(initial, plan, ctx.OutputDir))
		}
		if err := ConfirmCleanup(ctx, w.Yes, os.Stdin, os.Stdout); err != nil {
			ctx.ExitCode = app.ExitConfirmationRequired
			message := err.Error()
			if errors.Is(err, ErrConfirmationRequired) {
				message = "Real cleanup in non-interactive mode requires --yes"
			}
			result := app.OperationResult{
				Step:      "cleanup.confirmation",
				Target:    "user",
				Status:    app.OperationStatusFailed,
				Message:   message,
				Timestamp: time.Now(),
			}
			ctx.AddResult(result)
			logging.LogOperation(ctx.Logger, result)
			ctx.Logger.Warn("confirmation status: declined or missing")
			if !ctx.Quiet && !ctx.JSONOutput {
				fmt.Fprintln(os.Stderr, message)
			}
			ctx.JSONValue = CleanupExecutionResult{
				InitialDetection: initial,
				CleanupPlan:      plan,
				Operations:       ctx.Results,
				ExitCode:         ctx.ExitCode,
				ReportDir:        ctx.OutputDir,
			}
			_ = ctx.Reporter.WriteOperations(ctx.Results)
			_ = ctx.Reporter.WriteText("summary", FormatRealCleanupSummary(initial, nil, plan, ctx.Results, ctx))
			return nil
		}
		ctx.Logger.Info("confirmation status: accepted")
	}

	executor := NewExecutor(ctx.OutputDir, ctx.Logger)
	for _, result := range executor.ExecutePlan(plan) {
		ctx.AddResult(result)
	}

	final := detector.Detect()
	ctx.Logger.Info("final health: %s", final.Health)
	if err := ctx.Reporter.WriteJSON("final-detection", final); err != nil {
		return err
	}

	ctx.ExitCode = realCleanupExitCode(ctx.Results, final)
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)
	ctx.JSONValue = CleanupExecutionResult{
		InitialDetection: initial,
		CleanupPlan:      plan,
		Operations:       ctx.Results,
		FinalDetection:   &final,
		ExitCode:         ctx.ExitCode,
		ReportDir:        ctx.OutputDir,
	}
	summary := FormatRealCleanupSummary(initial, &final, plan, ctx.Results, ctx)
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

type CleanupExecutionResult struct {
	InitialDetection detector.DetectionReport  `json:"initial_detection"`
	CleanupPlan      CleanupPlan               `json:"cleanup_plan"`
	Operations       []app.OperationResult     `json:"operations"`
	FinalDetection   *detector.DetectionReport `json:"final_detection,omitempty"`
	ExitCode         int                       `json:"exit_code"`
	ReportDir        string                    `json:"report_dir"`
}

type CleanupPlanResult struct {
	Command          string                   `json:"command"`
	InitialDetection detector.DetectionReport `json:"initial_detection"`
	CleanupPlan      CleanupPlan              `json:"cleanup_plan"`
	Operations       []app.OperationResult    `json:"operations"`
	ExitCode         int                      `json:"exit_code"`
	ReportDir        string                   `json:"report_dir"`
	Warnings         []string                 `json:"warnings"`
	Errors           []string                 `json:"errors"`
}

func hasUnsafeAction(plan CleanupPlan) bool {
	for _, action := range plan.Actions {
		if !action.Safe {
			return true
		}
	}
	return false
}

func cleanupDryRunExitCode(plan CleanupPlan) int {
	switch {
	case len(plan.Blockers) > 0:
		return app.ExitInvalidInput
	case len(plan.Warnings) > 0:
		return app.ExitWarnings
	default:
		return app.ExitSuccess
	}
}

func realCleanupExitCode(results []app.OperationResult, final detector.DetectionReport) int {
	hasFailure := false
	hasWarning := false
	for _, result := range results {
		if !isExecutionResult(result) {
			continue
		}
		switch result.Status {
		case app.OperationStatusFailed:
			hasFailure = true
		case app.OperationStatusWarning:
			hasWarning = true
		}
	}
	leftovers := cleanupLeftoversRemain(final)
	switch {
	case hasFailure && leftovers:
		return app.ExitCleanupFailed
	case hasFailure || hasWarning:
		return app.ExitWarnings
	default:
		return app.ExitSuccess
	}
}

func cleanupLeftoversRemain(report detector.DetectionReport) bool {
	for _, service := range report.Services {
		if service.Exists {
			return true
		}
	}
	for _, process := range report.Processes {
		if process.GrabberRelated {
			return true
		}
	}
	for _, file := range report.Files {
		if file.Exists {
			return true
		}
	}
	for _, key := range report.Registry {
		if key.Exists {
			return true
		}
	}
	return false
}
