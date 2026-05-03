package cleaner

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/config"
	"kigrepair/internal/logging"
)

type CleanupWorkflow struct{}

func (w CleanupWorkflow) Name() string {
	return "cleanup"
}

func (w CleanupWorkflow) Run(ctx *app.AppContext) error {
	result := app.OperationResult{
		Step:      "cleanup.placeholder",
		Target:    "configured-cleanup-targets",
		Status:    app.OperationStatusSkipped,
		Message:   "Skeleton only: cleanup actions are not implemented.",
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
	return ctx.Reporter.WriteJSON("cleanup", map[string]any{
		"cleanup_paths": config.CleanupPaths,
		"note":          "No files, services, processes, registry keys, or MSI records were modified.",
	})
}
