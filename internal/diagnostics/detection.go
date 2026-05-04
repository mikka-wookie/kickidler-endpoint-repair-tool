package diagnostics

import (
	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type DetectionCollector struct {
	Report detector.DetectionReport
}

func (DetectionCollector) Name() string {
	return "detection"
}

func (c DetectionCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return writeJSON(ctx, "detection", c.Report)
}
