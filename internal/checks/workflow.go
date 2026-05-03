package checks

import (
	"fmt"
	"strings"
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
	ctx.Logger.Info("Starting detection-only check")
	report := detector.Detect()
	ctx.JSONValue = report

	status := app.OperationStatusSuccess
	if report.Health != detector.GrabberHealthHealthy {
		status = app.OperationStatusWarning
	}
	result := app.OperationResult{
		Step:      "check.detect",
		Target:    "grabber",
		Status:    status,
		Message:   "Detection completed with health: " + string(report.Health),
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
	if err := ctx.Reporter.WriteJSON("initial-detection", report); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote initial-detection.json")
	if err := ctx.Reporter.WriteText("summary", FormatSummary(report, ctx.OutputDir)); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote summary.txt")
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote operations.json")
	ctx.ExitCode = exitCodeForHealth(report.Health)
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatConsoleSummary(report, ctx.OutputDir))
	}
	return nil
}

func exitCodeForHealth(health detector.GrabberHealthStatus) int {
	switch health {
	case detector.GrabberHealthHealthy:
		return 0
	case detector.GrabberHealthNotInstalled:
		return 1
	case detector.GrabberHealthBroken:
		return 2
	case detector.GrabberHealthPartiallyRemoved:
		return 3
	default:
		return 4
	}
}

func FormatConsoleSummary(report detector.DetectionReport, reportDir string) string {
	return FormatSummary(report, reportDir)
}

func FormatSummary(report detector.DetectionReport, reportDir string) string {
	var b strings.Builder
	b.WriteString("Kigrepair Check Summary\n\n")
	b.WriteString("Health: " + string(report.Health) + "\n\n")
	if len(report.Issues) > 0 {
		b.WriteString("Issues:\n")
		for _, issue := range report.Issues {
			b.WriteString("- " + issue + "\n")
		}
		b.WriteString("\n")
	}
	if len(report.Recommendations) > 0 {
		b.WriteString("Recommendations:\n")
		for _, recommendation := range report.Recommendations {
			b.WriteString("- " + recommendation + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Report:\n")
	b.WriteString(reportDir)
	b.WriteString("\n")
	return b.String()
}
