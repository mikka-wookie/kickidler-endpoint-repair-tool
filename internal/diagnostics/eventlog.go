package diagnostics

import "kigrepair/internal/app"

type EventLogCollector struct{}

func (EventLogCollector) Name() string {
	return "eventlog"
}

func (EventLogCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return placeholderResult("collect.eventlog", "event-log")
}
