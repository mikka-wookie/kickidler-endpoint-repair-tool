package diagnostics

import (
	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type DefenderCollector struct {
	Report detector.DetectionReport
}

func (DefenderCollector) Name() string {
	return "defender"
}

func (c DefenderCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return writeJSON(ctx, "defender", c.Report.Defender)
}
