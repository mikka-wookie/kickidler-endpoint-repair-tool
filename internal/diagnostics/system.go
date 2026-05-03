package diagnostics

import "kigrepair/internal/app"

type SystemInfoCollector struct{}

func (SystemInfoCollector) Name() string {
	return "system"
}

func (SystemInfoCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.system", "system-info")
}
