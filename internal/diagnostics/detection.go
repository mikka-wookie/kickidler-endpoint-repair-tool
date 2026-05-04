package diagnostics

import (
	"path/filepath"

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
	if err := ctx.Reporter.WriteJSON("initial-detection", c.Report); err != nil {
		return operation("collect.detection", "initial-detection.json", app.OperationStatusFailed, "Failed to write initial-detection.json", err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, "initial-detection.json"))
	return operation("collect.detection", "initial-detection.json", app.OperationStatusSuccess, "Wrote initial-detection.json", "")
}
