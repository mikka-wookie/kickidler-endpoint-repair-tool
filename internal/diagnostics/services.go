package diagnostics

import "kigrepair/internal/app"

type ServicesCollector struct{}

func (ServicesCollector) Name() string {
	return "services"
}

func (ServicesCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.services", "services")
}
