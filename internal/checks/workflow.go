package checks

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
)

type CheckWorkflow struct{}

func (w CheckWorkflow) Name() string {
	return "check"
}

func (w CheckWorkflow) Run(ctx *app.AppContext) error {
	report := detector.Detect()
	result := app.OperationResult{
		Step:      "check.placeholder",
		Target:    "grabber",
		Status:    app.OperationStatusSkipped,
		Message:   "Skeleton only: real system checks are not implemented.",
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
	if err := ctx.Reporter.WriteJSON("check", report); err != nil {
		return err
	}
	return ctx.Reporter.WriteText("check-summary", "Check workflow placeholder. No system state was changed.\n")
}
