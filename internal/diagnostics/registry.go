package diagnostics

import (
	"path/filepath"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type RegistryCollector struct {
	Report detector.DetectionReport
}

func (RegistryCollector) Name() string {
	return "registry"
}

func (c RegistryCollector) Collect(ctx *app.AppContext) app.OperationResult {
	if err := ctx.Reporter.WriteJSON("system/registry", c.Report.Registry); err != nil {
		return operation("collect.registry", "system/registry.json", app.OperationStatusFailed, "Failed to write system/registry.json", err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, "system", "registry.json"))
	return operation("collect.registry", "system/registry.json", app.OperationStatusSuccess, "Wrote system/registry.json", "")
}
