package diagnostics

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/logging"
)

type Collector interface {
	Name() string
	Collect(ctx *app.AppContext) app.OperationResult
}

type CollectReportWorkflow struct{}

func (w CollectReportWorkflow) Name() string {
	return "collect-report"
}

func (w CollectReportWorkflow) Run(ctx *app.AppContext) error {
	collectors := []Collector{
		SystemInfoCollector{},
		ServicesCollector{},
		ProcessesCollector{},
		DefenderCollector{},
		RegistryCollector{},
		EventLogCollector{},
		NetworkCollector{},
	}
	for _, collector := range collectors {
		result := collector.Collect(ctx)
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
	}
	if err := ctx.Reporter.WriteText("collect-report", "Diagnostics collection placeholder. No archive or sensitive collection is performed yet.\n"); err != nil {
		return err
	}
	_, err := ctx.Reporter.Archive()
	return err
}

func placeholderResult(step string, target string) app.OperationResult {
	return app.OperationResult{
		Step:      step,
		Target:    target,
		Status:    app.OperationStatusSkipped,
		Message:   "Skeleton only: collection is not implemented.",
		Timestamp: time.Now(),
	}
}
