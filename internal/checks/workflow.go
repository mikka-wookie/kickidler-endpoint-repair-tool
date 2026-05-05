package checks

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/classifier"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/recommendations"
	"kigrepair/internal/reports"
)

type CheckWorkflow struct{}

func (w CheckWorkflow) Name() string {
	return "check"
}

func (w CheckWorkflow) Run(ctx *app.AppContext) error {
	ctx.Logger.Info("Starting detection-only check")
	report := detector.Detect()
	logDefenderDetection(ctx.Logger, report)
	classification := classifier.Classify(classifier.ClassificationInput{Detection: &report})

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
	recommendation := recommendations.Plan(recommendations.RecommendationInput{
		Detection:      &report,
		Classification: ptr(recommendations.FromClassifier(classification)),
		IsAdmin:        report.IsAdmin,
		IsInteractive:  !ctx.NonInteractive && !ctx.Quiet,
		OutputDir:      ctx.OutputDir,
	})
	if err := ctx.Reporter.WriteJSON("classification-result", classification); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote classification-result.json")
	if err := ctx.Reporter.WriteJSON("recommendation-result", recommendation); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote recommendation-result.json")
	ctx.ExitCode = exitCodeForHealth(report.Health)
	ctx.JSONValue = CheckResult{
		Command:        "check",
		ReportDir:      ctx.OutputDir,
		ExitCode:       ctx.ExitCode,
		Warnings:       report.Recommendations,
		Errors:         report.Issues,
		Detection:      report,
		Recommendation: recommendation,
		Classification: classification,
	}
	if err := ctx.Reporter.WriteText("summary", FormatSummary(report, classification, ctx, recommendation)); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote summary.txt")
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote operations.json")
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatConsoleSummary(report, classification, ctx, recommendation))
	}
	return nil
}

type CheckResult struct {
	Command        string                               `json:"command"`
	ReportDir      string                               `json:"report_dir"`
	ExitCode       int                                  `json:"exit_code"`
	Warnings       []string                             `json:"warnings"`
	Errors         []string                             `json:"errors"`
	Detection      detector.DetectionReport             `json:"detection"`
	Recommendation recommendations.RecommendationResult `json:"recommendation"`
	Classification classifier.ClassificationResult      `json:"classification"`
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

func FormatConsoleSummary(report detector.DetectionReport, classification classifier.ClassificationResult, ctx *app.AppContext, recommendation recommendations.RecommendationResult) string {
	return FormatSummary(report, classification, ctx, recommendation)
}

func FormatSummary(report detector.DetectionReport, classification classifier.ClassificationResult, ctx *app.AppContext, recommendation recommendations.RecommendationResult) string {
	var b strings.Builder
	b.WriteString("Kigrepair Check Summary\n\n")
	b.WriteString("Health: " + string(report.Health) + "\n")
	b.WriteString("Install mode: " + string(report.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(report.InstallRoot) + "\n")
	b.WriteString("Primary service: " + valueOrDash(report.PrimaryService) + "\n")
	b.WriteString("Service status: " + serviceStatus(report) + "\n")
	b.WriteString("Service executable: " + valueOrDash(report.ServiceExecutablePath) + "\n")
	b.WriteString("Defender exclusion: " + defenderExclusionStatus(report) + "\n\n")
	b.WriteString("Processes: " + processStatusLine(report) + "\n\n")
	if coveredBy := defenderCoveredBy(report); coveredBy != "" {
		b.WriteString("Covered by: " + coveredBy + "\n\n")
	}
	b.WriteString(classifier.FormatSection(classification))
	b.WriteString("\n")
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
	b.WriteString(recommendations.FormatConsole(recommendation))
	b.WriteString("\n")
	b.WriteString("Report:\n")
	b.WriteString(ctx.OutputDir)
	b.WriteString("\n")
	return reports.FormatSummary(reports.SummaryData{
		Command:        "check",
		Started:        ctx.StartedAt,
		Finished:       time.Now(),
		Mode:           string(ctx.Mode),
		ExitCode:       ctx.ExitCode,
		ReportDir:      ctx.OutputDir,
		InitialHealth:  string(report.Health),
		InstallMode:    string(report.InstallMode),
		InstallRoot:    report.InstallRoot,
		PrimaryService: report.PrimaryService,
		Warnings:       report.Recommendations,
		Errors:         report.Issues,
		Actions:        []string{"Detection completed with health: " + string(report.Health)},
	}, b.String()+recommendations.FormatSection(recommendation))
}

func processStatusLine(report detector.DetectionReport) string {
	trusted := 0
	skipped := 0
	for _, process := range report.Processes {
		if detector.TrustedProcessForTermination(process) {
			trusted++
			continue
		}
		if len(process.Warnings) > 0 || process.TrustLevel == detector.ProcessTrustPathMismatch || process.TrustLevel == detector.ProcessTrustPathUnavailable {
			skipped++
		}
	}
	return fmt.Sprintf("%d trusted Grabber processes detected, %d skipped unsafe name match.", trusted, skipped)
}

func ptr[T any](value T) *T {
	return &value
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func serviceStatus(report detector.DetectionReport) string {
	if report.PrimaryService == "" {
		return "-"
	}
	for _, service := range report.Services {
		if strings.EqualFold(service.Name, report.PrimaryService) {
			if service.Status != "" {
				return service.Status
			}
			if service.Exists {
				return "exists"
			}
		}
	}
	return "-"
}

func defenderExclusionStatus(report detector.DetectionReport) string {
	if !report.Defender.Available {
		return "unavailable"
	}
	if len(report.MissingDefenderPaths) > 0 {
		return "missing"
	}
	if len(report.RequiredDefenderPaths) > 0 {
		return "present"
	}
	return "-"
}

func defenderCoveredBy(report detector.DetectionReport) string {
	if !report.Defender.Available {
		return ""
	}
	for _, requiredPath := range report.RequiredDefenderPaths {
		requiredNormalized := detector.NormalizeWindowsPath(requiredPath)
		for _, exclusionPath := range report.Defender.ExclusionPaths {
			if detector.IsPathCoveredByExclusion(requiredPath, exclusionPath) && !strings.EqualFold(requiredNormalized, detector.NormalizeWindowsPath(exclusionPath)) {
				return exclusionPath
			}
		}
	}
	return ""
}

func logDefenderDetection(logger app.Logger, report detector.DetectionReport) {
	logger.Info("Defender raw exclusion paths: %v", report.Defender.ExclusionPaths)
	logger.Info("Defender normalized exclusion paths: %v", report.Defender.NormalizedExclusionPaths)
	logger.Info("Defender required paths: %v", report.Defender.RequiredPaths)
	logger.Info("Defender covered paths: %v", report.Defender.CoveredPaths)
	logger.Info("Defender missing paths: %v", report.Defender.MissingPaths)
	if report.Defender.Error != "" {
		logger.Warn("Defender detection error: %s", report.Defender.Error)
	}
}
