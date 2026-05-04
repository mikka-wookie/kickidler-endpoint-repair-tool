package diagnostics

import "kigrepair/internal/app"

type NetworkCollector struct{}

func (NetworkCollector) Name() string {
	return "network"
}

func (NetworkCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return operation("collect.network", "network", app.OperationStatusSkipped, "Network collection is not implemented", "")
}
