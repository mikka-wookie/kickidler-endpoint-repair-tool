package diagnostics

import (
	"os"
	"path/filepath"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/reports"
)

type testCollector struct {
	name   string
	status app.OperationStatus
}

func (c testCollector) Name() string {
	return c.name
}

func (c testCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return operation("collect."+c.name, c.name, c.status, c.name+" done", "")
}

func TestCollectReportWorkflowContinuesAfterFailedCollector(t *testing.T) {
	reportDir := t.TempDir()
	ctx := app.NewContext()
	ctx.OutputDir = reportDir
	ctx.ReportRoot = filepath.Dir(reportDir)
	ctx.Logger = logging.Discard()
	reporter, err := reports.New(reportDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Reporter = reporter

	workflow := CollectReportWorkflow{
		NoZip: true,
		Detect: func() detector.DetectionReport {
			return detector.DetectionReport{Health: detector.GrabberHealthUnknown, InstallMode: detector.InstallModeUnknown}
		},
		Collectors: []Collector{
			testCollector{name: "first", status: app.OperationStatusFailed},
			testCollector{name: "second", status: app.OperationStatusSuccess},
		},
	}

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(ctx.Results) < 3 {
		t.Fatalf("expected collector and zip results, got %#v", ctx.Results)
	}
	if ctx.Results[0].Step != "collect.first" || ctx.Results[1].Step != "collect.second" {
		t.Fatalf("collectors did not run in order: %#v", ctx.Results)
	}
	if ctx.ExitCode != 1 {
		t.Fatalf("ExitCode = %d", ctx.ExitCode)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "collect-result.json")); err != nil {
		t.Fatalf("collect-result.json missing: %v", err)
	}
}
