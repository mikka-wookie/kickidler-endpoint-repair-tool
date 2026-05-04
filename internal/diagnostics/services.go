package diagnostics

import (
	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type ServicesCollector struct {
	Report detector.DetectionReport
}

func (ServicesCollector) Name() string {
	return "services"
}

func (c ServicesCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return writeJSON(ctx, "services", c.Report.Services)
}
