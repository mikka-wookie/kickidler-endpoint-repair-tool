package installer

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/logging"
)

type InstallWorkflow struct {
	Invite    string `json:"invite,omitempty"`
	Installer string `json:"installer,omitempty"`
}

func (w InstallWorkflow) Name() string {
	return "install"
}

func (w InstallWorkflow) Run(ctx *app.AppContext) error {
	result := app.OperationResult{
		Step:      "install.placeholder",
		Target:    "grabber",
		Status:    app.OperationStatusSkipped,
		Message:   "Skeleton only: installer execution is not implemented.",
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
	return ctx.Reporter.WriteJSON("install", w)
}
