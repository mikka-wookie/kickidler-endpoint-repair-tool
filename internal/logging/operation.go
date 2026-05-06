package logging

import "kigrepair/internal/app"

func LogOperation(logger app.Logger, result app.OperationResult) {
	if logger == nil {
		return
	}
	if structured, ok := logger.(interface{ Event(Event) }); ok {
		level := LevelInfo
		switch result.Status {
		case app.OperationStatusFailed:
			level = LevelError
		case app.OperationStatusWarning:
			level = LevelWarning
		}
		structured.Event(Event{
			Level:           level,
			RunID:           result.RunID,
			Workflow:        result.Workflow,
			OperationID:     result.ID,
			Event:           "operation_finished",
			Message:         result.Message,
			Category:        result.Category,
			Status:          string(result.Status),
			DurationMS:      result.DurationMS,
			FailureCategory: result.FailureCategory,
			Fields: map[string]any{
				"step":             result.Step,
				"target":           result.Target,
				"error":            result.Error,
				"redacted_command": result.RedactedCommand,
				"result_file":      result.ResultFile,
				"related_files":    result.RelatedFiles,
				"read_only":        result.ReadOnly,
				"dry_run":          result.DryRun,
			},
		})
		return
	}
	switch result.Status {
	case app.OperationStatusFailed:
		logger.Error("%s %s: %s", result.Step, result.Target, result.Message)
	case app.OperationStatusWarning:
		logger.Warn("%s %s: %s", result.Step, result.Target, result.Message)
	default:
		logger.Info("%s %s: %s", result.Step, result.Target, result.Message)
	}
}
