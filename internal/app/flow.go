package app

func RunWorkflow(ctx *AppContext, workflow Workflow) error {
	if ctx.Logger != nil {
		ctx.Logger.Info("starting workflow: %s", workflow.Name())
	}

	err := workflow.Run(ctx)
	if writeErr := ctx.Reporter.WriteOperations(ctx.Results); writeErr != nil && err == nil {
		err = writeErr
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
