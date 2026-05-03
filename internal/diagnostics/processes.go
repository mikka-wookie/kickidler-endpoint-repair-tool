package diagnostics

import "kigrepair/internal/app"

type ProcessesCollector struct{}

func (ProcessesCollector) Name() string {
	return "processes"
}

func (ProcessesCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.processes", "processes")
}
