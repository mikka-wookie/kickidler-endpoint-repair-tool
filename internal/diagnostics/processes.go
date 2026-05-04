package diagnostics

import (
	"path/filepath"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type ProcessesCollector struct {
	Report detector.DetectionReport
}

func (ProcessesCollector) Name() string {
	return "processes"
}

func (c ProcessesCollector) Collect(ctx *app.AppContext) app.OperationResult {
	if err := ctx.Reporter.WriteJSON("system/processes", c.Report.Processes); err != nil {
		return operation("collect.processes", "system/processes.json", app.OperationStatusFailed, "Failed to write system/processes.json", err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, "system", "processes.json"))
	return operation("collect.processes", "system/processes.json", app.OperationStatusSuccess, "Wrote system/processes.json", "")
}
