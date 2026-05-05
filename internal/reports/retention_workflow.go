package reports

import (
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/logging"
)

type CleanupReportsWorkflow struct {
	OlderThan string
	KeepLast  int
	DryRun    bool
	Yes       bool
}

func (w CleanupReportsWorkflow) Name() string {
	return "reports cleanup"
}

func (w CleanupReportsWorkflow) Run(ctx *app.AppContext) error {
	ctx.Logger.Info("report cleanup started")
	ctx.Logger.Info("reports root: %s", ctx.ReportRoot)
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)
	plan, err := BuildReportCleanupPlan(ctx.ReportRoot, ReportCleanupOptions{
		OlderThan:       w.OlderThan,
		KeepLast:        w.KeepLast,
		DryRun:          w.DryRun,
		Yes:             w.Yes,
		ActiveReportDir: ctx.OutputDir,
	})
	if err != nil {
		ctx.Logger.Warn("report cleanup plan build warning: %s", err)
	}
	addReportOperation(ctx, "reports.cleanup.plan_build", ctx.ReportRoot, operationStatusForPlan(plan), "Report cleanup plan built", strings.Join(plan.Errors, "; "))
	for _, target := range plan.PlannedDeletes {
		addReportOperation(ctx, "reports.cleanup.target_validation", target.Path, app.OperationStatusSuccess, "Report cleanup target validated", "")
	}
	for _, skipped := range plan.Skipped {
		addReportOperation(ctx, "reports.cleanup.target_validation", skipped.Path, app.OperationStatusSkipped, skipped.Reason, "")
	}

	result := ExecuteReportCleanupPlan(ctx.ReportRoot, plan, w.Yes)
	result.ReportDir = ctx.OutputDir
	result.PlanPath = filepath.Join(ctx.OutputDir, "report-cleanup-plan.json")
	result.PlannedDeletes = len(plan.PlannedDeletes)
	result.DeletedCount = countDeleteStatus(result.Deleted, "success")
	result.Warnings = uniqueReportStrings(append(result.Warnings, plan.Warnings...))
	result.Errors = uniqueReportStrings(append(result.Errors, plan.Errors...))
	if len(result.Errors) > 0 {
		result.Status = "failed"
		result.ExitCode = 10
	} else if len(result.Warnings) > 0 && result.Status == "success" {
		result.Status = "warning"
		result.ExitCode = 1
	}
	if !w.DryRun && !w.Yes {
		result.Status = "failed"
		result.ExitCode = 7
	}
	ctx.ExitCode = result.ExitCode
	ctx.JSONValue = result

	status := app.OperationStatusSuccess
	if result.Status == "warning" {
		status = app.OperationStatusWarning
	}
	if result.Status == "failed" {
		status = app.OperationStatusFailed
	}
	addReportOperation(ctx, "reports.cleanup.execution", ctx.ReportRoot, status, "Report cleanup execution completed", strings.Join(result.Errors, "; "))

	if err := ctx.Reporter.WriteJSON("report-cleanup-plan", plan); err != nil {
		return err
	}
	addReportOperation(ctx, "reports.cleanup.result_writing", filepath.Join(ctx.OutputDir, "report-cleanup-plan.json"), app.OperationStatusSuccess, "Report cleanup plan written", "")
	if err := ctx.Reporter.WriteJSON("report-cleanup-result", result); err != nil {
		return err
	}
	addReportOperation(ctx, "reports.cleanup.result_writing", filepath.Join(ctx.OutputDir, "report-cleanup-result.json"), app.OperationStatusSuccess, "Report cleanup result written", "")
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	summary := FormatReportCleanupSummary(plan, result, ctx.StartedAt)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		printReportSummary(summary)
	}
	return nil
}

func addReportOperation(ctx *app.AppContext, step string, target string, status app.OperationStatus, message string, errText string) {
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

func operationStatusForPlan(plan ReportCleanupPlan) app.OperationStatus {
	if len(plan.Errors) > 0 {
		return app.OperationStatusFailed
	}
	if len(plan.Warnings) > 0 {
		return app.OperationStatusWarning
	}
	return app.OperationStatusSuccess
}
