package diagnostics

import (
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
	return writeJSON(ctx, "processes", c.Report.Processes)
}
