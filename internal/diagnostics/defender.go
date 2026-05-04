package diagnostics

import (
	"path/filepath"

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
	if err := ctx.Reporter.WriteJSON("system/defender", c.Report.Defender); err != nil {
		return operation("collect.defender", "system/defender.json", app.OperationStatusFailed, "Failed to write system/defender.json", err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, "system", "defender.json"))
	return operation("collect.defender", "system/defender.json", app.OperationStatusSuccess, "Wrote system/defender.json", "")
}
