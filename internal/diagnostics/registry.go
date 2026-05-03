package diagnostics

import "kigrepair/internal/app"

type RegistryCollector struct{}

func (RegistryCollector) Name() string {
	return "registry"
}

func (RegistryCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.registry", "registry")
}
