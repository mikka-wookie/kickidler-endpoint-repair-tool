package diagnostics

import "kigrepair/internal/app"

type DefenderCollector struct{}

func (DefenderCollector) Name() string {
	return "defender"
}

func (DefenderCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.defender", "defender")
}
