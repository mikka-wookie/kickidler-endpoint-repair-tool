package repair

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/logging"
)

type RepairWorkflow struct {
	Invite    string `json:"invite,omitempty"`
	Installer string `json:"installer,omitempty"`
}

func (w RepairWorkflow) Name() string {
	return "repair"
}

func (w RepairWorkflow) Run(ctx *app.AppContext) error {
	result := app.OperationResult{
		Step:      "repair.placeholder",
		Target:    "grabber",
		Status:    app.OperationStatusSkipped,
		Message:   "Skeleton only: repair/reinstall actions are not implemented.",
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
	return ctx.Reporter.WriteJSON("repair", w)
}
