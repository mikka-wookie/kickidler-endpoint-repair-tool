package defender

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/logging"
)

type DefenderWorkflow struct {
	Ensure bool `json:"ensure"`
}

func (w DefenderWorkflow) Name() string {
	return "defender"
}

func (w DefenderWorkflow) Run(ctx *app.AppContext) error {
	result := app.OperationResult{
		Step:      "defender.placeholder",
		Target:    "defender-exclusions",
		Status:    app.OperationStatusSkipped,
		Message:   "Skeleton only: Defender exclusions are not modified.",
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
	return ctx.Reporter.WriteJSON("defender", w)
}
