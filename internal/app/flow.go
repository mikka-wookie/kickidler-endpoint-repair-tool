package app

import "kigrepair/internal/failures"

func RunWorkflow(ctx *AppContext, workflow Workflow) error {
	ctx.Run.WorkflowName = workflow.Name()
	ctx.Run.WorkflowID = ctx.Run.RunID + "-" + Slug(workflow.Name())
	if ctx.Run.CommandName == "" {
		ctx.Run.CommandName = workflow.Name()
	}
	ctx.Run.ReportDir = ctx.OutputDir
	ctx.Run.UserInteractive = !ctx.Quiet && !ctx.NonInteractive
	if ctx.Logger != nil {
		ctx.Logger.Info("starting workflow: %s", workflow.Name())
	}

	err := workflow.Run(ctx)
	ctx.FinishRun()
	if ctx.Reporter != nil {
		if writeErr := ctx.Reporter.WriteOperations(ctx.Results); writeErr != nil && err == nil {
			ctx.AddResult(OperationResult{
				Step:     "report_write",
				Target:   "operations.json",
				Status:   OperationStatusWarning,
				Message:  "Failed to write operations report",
				Error:    writeErr.Error(),
				Category: failures.FailureReportWrite,
			})
			err = writeErr
		}
	} else if err == nil {
		ctx.AddResult(OperationResult{
			Step:     "report_write",
			Target:   "operations.json",
			Status:   OperationStatusWarning,
			Message:  "Reporter is unavailable; operations report was not written",
			Error:    "reporter is nil",
			Category: failures.FailureReportWrite,
		})
	}

	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Error("workflow failed: %s: %v", workflow.Name(), err)
		}
		return err
	}

	if ctx.Logger != nil {
		ctx.Logger.Info("workflow completed: %s", workflow.Name())
	}
	return nil
}
