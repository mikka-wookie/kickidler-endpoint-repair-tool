package diagnostics

import (
	"path/filepath"
	"strings"

	"kigrepair/internal/app"
)

type EventLogCollector struct{}

func (EventLogCollector) Name() string {
	return "eventlogs"
}

func (EventLogCollector) Collect(ctx *app.AppContext) app.OperationResult {
	dir := filepath.Join(ctx.OutputDir, "eventlogs")
	if err := ensureDir(dir); err != nil {
		return operation("collect.eventlogs", "eventlogs", app.OperationStatusFailed, "Failed to create eventlogs directory", err.Error())
	}
	content := strings.Join([]string{
		"Event log collection is not implemented yet.",
		"",
		"This placeholder is intentionally read-only. A later implementation should query a limited",
		"set of recent Application, System, and Defender Operational events without exporting",
		"large logs or unrelated channels.",
		"",
	}, "\n")
	if err := writeTextFile(filepath.Join(dir, "README.txt"), content); err != nil {
		return operation("collect.eventlogs", "eventlogs", app.OperationStatusFailed, "Failed to write event log placeholder", err.Error())
	}
	return operation("collect.eventlogs", "eventlogs", app.OperationStatusWarning, "Wrote eventlogs/README.txt; event log collection is not implemented yet", "event log collection is not implemented yet")
}
