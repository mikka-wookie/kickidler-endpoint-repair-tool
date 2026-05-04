package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/reports"
)

type Collector interface {
	Name() string
	Collect(ctx *app.AppContext) app.OperationResult
}

type CollectReportWorkflow struct {
	IncludeEventLogs bool
	IncludeHistory   bool
	HistoryLimit     int
	NoZip            bool

	Detect     func() detector.DetectionReport
	Collectors []Collector
}

func (w CollectReportWorkflow) Name() string {
	return "collect-report"
}

func (w CollectReportWorkflow) Run(ctx *app.AppContext) error {
	startedAt := time.Now()
	result := CollectReportResult{
		StartedAt:        startedAt,
		Mode:             string(ctx.Mode),
		ReportDir:        ctx.OutputDir,
		Collectors:       []CollectorResult{},
		FilesIncluded:    []CollectedFile{},
		FilesSkipped:     []string{},
		Warnings:         []string{},
		Errors:           []string{},
		RedactionEnabled: true,
	}
	ctx.JSONValue = result
	ctx.Logger.Info("collect-report started")
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	detection := detect()
	result.Health = string(detection.Health)
	result.InstallMode = string(detection.InstallMode)
	result.InstallRoot = detection.InstallRoot
	ctx.Logger.Info("admin status: %t", detection.IsAdmin)
	ctx.Logger.Info("detected health/mode/root: %s / %s / %s", detection.Health, detection.InstallMode, detection.InstallRoot)

	collectors := w.collectors(detection)
	for _, collector := range collectors {
		ctx.Logger.Info("collector start: %s", collector.Name())
		result := collector.Collect(ctx)
		ctx.AddResult(result)
		logging.LogOperation(ctx.Logger, result)
		ctx.Logger.Info("collector end: %s status=%s", collector.Name(), result.Status)
	}

	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote operations.json")

	result.Collectors = collectorResults(ctx.Results)
	for _, collector := range result.Collectors {
		switch collector.Status {
		case string(app.OperationStatusFailed):
			result.Warnings = append(result.Warnings, collector.Name+": "+firstNonEmpty(collector.Error, collector.Message))
		case string(app.OperationStatusWarning):
			result.Warnings = append(result.Warnings, collector.Name+": "+collector.Message)
		}
	}

	if !w.NoZip {
		result.BundlePath = filepath.Join(ctx.OutputDir, reports.SupportBundleName)
	}
	result.FinishedAt = time.Now()
	ctx.JSONValue = result
	if err := ctx.Reporter.WriteJSON("collect-result", result); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote collect-result.json")
	if err := ctx.Reporter.WriteText("summary", FormatCollectSummary(result)); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote summary.txt")

	if !w.NoZip {
		ctx.Logger.Info("collector inspect: zip")
		archive := reports.InspectSupportBundle(ctx.OutputDir)
		status := app.OperationStatusSuccess
		message := "Support bundle created"
		errorText := ""
		result.BundlePath = archive.Path
		result.Warnings = append(result.Warnings, archive.Warnings...)
		result.Errors = append(result.Errors, archive.Errors...)
		result.FilesSkipped = append(result.FilesSkipped, archive.Skipped...)
		result.FilesIncluded = collectedFiles(archive.Included)
		if len(archive.Warnings) > 0 {
			status = app.OperationStatusWarning
			message = "Support bundle created with warnings"
			errorText = strings.Join(archive.Warnings, "; ")
		}
		if len(archive.Errors) > 0 {
			status = app.OperationStatusFailed
			message = "Support bundle created with errors"
			errorText = strings.Join(archive.Errors, "; ")
		}
		zipOperation := operation("collect.zip", "zip", status, message, errorText)
		ctx.AddResult(zipOperation)
		logging.LogOperation(ctx.Logger, zipOperation)
		ctx.Logger.Info("collector inspect end: zip status=%s", status)
	} else {
		zipOperation := operation("collect.zip", "zip", app.OperationStatusSkipped, "ZIP creation skipped by --no-zip", "")
		ctx.AddResult(zipOperation)
		logging.LogOperation(ctx.Logger, zipOperation)
	}

	result.Collectors = collectorResults(ctx.Results)
	result.Status = collectStatus(result)
	ctx.ExitCode = exitCode(result)
	result.ExitCode = ctx.ExitCode
	result.FinishedAt = time.Now()
	ctx.JSONValue = result

	if err := ctx.Reporter.WriteJSON("collect-result", result); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote collect-result.json")
	if err := ctx.Reporter.WriteText("summary", FormatCollectSummary(result)); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote summary.txt")
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	ctx.Logger.Info("Wrote operations.json")

	if !w.NoZip {
		ctx.Logger.Info("collector start: zip")
		archive, err := reports.CreateSupportBundle(ctx.OutputDir)
		if err != nil {
			errorText := err.Error()
			result.BundlePath = ""
			result.Warnings = append(result.Warnings, errorText)
			result.Status = collectStatus(result)
			ctx.ExitCode = exitCode(result)
			result.ExitCode = ctx.ExitCode
			ctx.JSONValue = result
			if writeErr := ctx.Reporter.WriteJSON("collect-result", result); writeErr != nil {
				return writeErr
			}
			if writeErr := ctx.Reporter.WriteText("summary", FormatCollectSummary(result)); writeErr != nil {
				return writeErr
			}
			ctx.Logger.Warn("support bundle creation failed: %s", errorText)
		} else {
			ctx.Logger.Info("ZIP path: %s", archive.Path)
		}
		ctx.Logger.Info("collector end: zip")
	}
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)

	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatCollectSummary(result))
	}
	return nil
}

func collectedFiles(files []reports.BundleFile) []CollectedFile {
	result := make([]CollectedFile, 0, len(files))
	for _, file := range files {
		result = append(result, CollectedFile{
			Path:      file.Path,
			Source:    file.Source,
			SizeBytes: file.SizeBytes,
			SHA256:    file.SHA256,
		})
	}
	return result
}

func collectStatus(result CollectReportResult) string {
	if len(result.Errors) > 0 {
		return "failed"
	}
	if len(result.Warnings) > 0 {
		return "warning"
	}
	for _, collector := range result.Collectors {
		if collector.Status == string(app.OperationStatusFailed) {
			return "failed"
		}
		if collector.Status == string(app.OperationStatusWarning) {
			return "warning"
		}
	}
	return "success"
}

func (w CollectReportWorkflow) collectors(detection detector.DetectionReport) []Collector {
	if w.Collectors != nil {
		return w.Collectors
	}
	historyLimit := w.HistoryLimit
	if historyLimit <= 0 {
		historyLimit = 5
	}
	collectors := []Collector{
		DetectionCollector{Report: detection},
		SystemInfoCollector{Report: detection},
		ServicesCollector{Report: detection},
		ProcessesCollector{Report: detection},
		DefenderCollector{Report: detection},
		RegistryCollector{Report: detection},
	}
	if w.IncludeEventLogs {
		collectors = append(collectors, EventLogCollector{})
	} else {
		collectors = append(collectors, SkippedCollector{NameValue: "eventlogs", StepValue: "collect.eventlogs", TargetValue: "eventlogs", MessageValue: "Event log collection disabled"})
	}
	if w.IncludeHistory {
		collectors = append(collectors, HistoryCollector{Limit: historyLimit}, MSILogsCollector{Limit: historyLimit})
	} else {
		collectors = append(collectors,
			SkippedCollector{NameValue: "history", StepValue: "collect.history", TargetValue: "history", MessageValue: "History collection disabled"},
			SkippedCollector{NameValue: "msi-logs", StepValue: "collect.msi_logs", TargetValue: "msi-logs", MessageValue: "MSI log collection disabled"},
		)
	}
	return collectors
}

type SkippedCollector struct {
	NameValue    string
	StepValue    string
	TargetValue  string
	MessageValue string
}

func (c SkippedCollector) Name() string {
	return c.NameValue
}

func (c SkippedCollector) Collect(ctx *app.AppContext) app.OperationResult {
	return operation(c.StepValue, c.TargetValue, app.OperationStatusSkipped, c.MessageValue, "")
}

func operation(step string, target string, status app.OperationStatus, message string, errorText string) app.OperationResult {
	return app.OperationResult{
		Step:      step,
		Target:    target,
		Status:    status,
		Message:   message,
		Error:     errorText,
		Timestamp: time.Now(),
	}
}

func writeJSON(ctx *app.AppContext, name string, value any) app.OperationResult {
	if err := ctx.Reporter.WriteJSON(name, value); err != nil {
		return operation("collect."+strings.TrimSuffix(name, ".json"), name, app.OperationStatusFailed, "Failed to write "+name, err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, ensureExt(name, ".json")))
	return operation("collect."+strings.TrimSuffix(name, ".json"), name, app.OperationStatusSuccess, "Wrote "+ensureExt(name, ".json"), "")
}

func writeTextFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func collectorResults(results []app.OperationResult) []CollectorResult {
	collectors := make([]CollectorResult, 0, len(results))
	for _, result := range results {
		if !strings.HasPrefix(result.Step, "collect.") {
			continue
		}
		name := strings.TrimPrefix(result.Step, "collect.")
		name = strings.ReplaceAll(name, "_", "-")
		collectors = append(collectors, CollectorResult{
			Name:        name,
			Status:      string(result.Status),
			OutputFiles: outputFilesFromResult(name, result),
			Message:     result.Message,
			Error:       result.Error,
		})
	}
	return collectors
}

func outputFilesFromResult(name string, result app.OperationResult) []string {
	switch name {
	case "detection":
		return []string{"initial-detection.json"}
	case "system":
		return []string{filepath.ToSlash(filepath.Join("system", "environment.json"))}
	case "services":
		return []string{filepath.ToSlash(filepath.Join("system", "services.json"))}
	case "processes":
		return []string{filepath.ToSlash(filepath.Join("system", "processes.json"))}
	case "defender":
		return []string{filepath.ToSlash(filepath.Join("system", "defender.json"))}
	case "registry":
		return []string{filepath.ToSlash(filepath.Join("system", "registry.json"))}
	case "eventlogs":
		if result.Status == app.OperationStatusSuccess || result.Status == app.OperationStatusWarning {
			return []string{filepath.ToSlash(filepath.Join("eventlogs", "README.txt"))}
		}
	case "history":
		if result.Status == app.OperationStatusSuccess || result.Status == app.OperationStatusWarning {
			return []string{"history"}
		}
	case "msi-logs":
		if result.Status == app.OperationStatusSuccess || result.Status == app.OperationStatusWarning {
			return []string{"msi-logs"}
		}
	case "zip":
		if result.Status == app.OperationStatusSuccess || result.Status == app.OperationStatusWarning {
			return []string{reports.SupportBundleName}
		}
	}
	return nil
}

func exitCode(result CollectReportResult) int {
	if len(result.Errors) > 0 {
		return 10
	}
	if len(result.Warnings) > 0 {
		return 1
	}
	for _, collector := range result.Collectors {
		if collector.Status == string(app.OperationStatusFailed) || collector.Status == string(app.OperationStatusWarning) {
			return 1
		}
	}
	return 0
}

func ensureExt(name string, ext string) string {
	if strings.EqualFold(filepath.Ext(name), ext) {
		return name
	}
	return name + ext
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
