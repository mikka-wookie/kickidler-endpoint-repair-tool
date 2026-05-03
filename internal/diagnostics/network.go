package diagnostics

import "kigrepair/internal/app"

type NetworkCollector struct{}

func (NetworkCollector) Name() string {
	return "network"
}

func (NetworkCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.network", "network")
}
