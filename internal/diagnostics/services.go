package diagnostics

import (
	"path/filepath"

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
	if err := ctx.Reporter.WriteJSON("system/services", c.Report.Services); err != nil {
		return operation("collect.services", "system/services.json", app.OperationStatusFailed, "Failed to write system/services.json", err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, "system", "services.json"))
	return operation("collect.services", "system/services.json", app.OperationStatusSuccess, "Wrote system/services.json", "")
}
