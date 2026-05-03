package logging

import "kigrepair/internal/app"

func LogOperation(logger app.Logger, result app.OperationResult) {
	if logger == nil {
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
