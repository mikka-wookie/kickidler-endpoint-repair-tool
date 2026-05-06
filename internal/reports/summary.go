package reports

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/safety"
)

type SummaryData struct {
	Command        string
	Started        time.Time
	Finished       time.Time
	Mode           string
	ExitCode       int
	ReportDir      string
	InitialHealth  string
	FinalHealth    string
	InstallMode    string
	InstallRoot    string
	PrimaryService string
	Warnings       []string
	Errors         []string
	Actions        []string
	Run            app.RunMetadata
	Operations     []app.OperationResult
}

func FormatSummary(data SummaryData, details string) string {
	finished := data.Finished
	if finished.IsZero() {
		finished = time.Now()
	}
	var b strings.Builder
	if data.Run.RunID != "" {
		b.WriteString("Run ID: " + data.Run.RunID + "\n")
		b.WriteString("Correlation ID: " + valueOrDash(data.Run.CorrelationID) + "\n")
		b.WriteString("Workflow: " + valueOrDash(data.Run.WorkflowName) + "\n")
	}
	b.WriteString("Command: " + valueOrDash(data.Command) + "\n")
	b.WriteString("Started: " + formatTime(data.Started) + "\n")
	b.WriteString("Finished: " + formatTime(finished) + "\n")
	if !data.Started.IsZero() {
		b.WriteString("Duration: " + finished.Sub(data.Started).Round(time.Millisecond).String() + "\n")
	} else {
		b.WriteString("Duration: -\n")
	}
	b.WriteString("Mode: " + valueOrDash(data.Mode) + "\n")
	b.WriteString(fmt.Sprintf("Exit code: %d\n", data.ExitCode))
	b.WriteString("Report directory: " + valueOrDash(data.ReportDir) + "\n")
	writeOptional(&b, "Initial health", data.InitialHealth)
	writeOptional(&b, "Final health", data.FinalHealth)
	writeOptional(&b, "Install mode", data.InstallMode)
	writeOptional(&b, "Install root", data.InstallRoot)
	writeOptional(&b, "Primary service", data.PrimaryService)
	b.WriteString("\n")
	writeList(&b, "Warnings", data.Warnings)
	writeList(&b, "Errors", data.Errors)
	writeList(&b, "Main actions/results", data.Actions)
	writeTimeline(&b, data.Operations)
	if strings.TrimSpace(details) != "" {
		b.WriteString(safety.RedactString(details))
		if !strings.HasSuffix(details, "\n") {
			b.WriteString("\n")
		}
	}
	return safety.RedactString(b.String())
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format(time.RFC3339)
}

func writeOptional(b *strings.Builder, name string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	b.WriteString(name + ": " + value + "\n")
}

func writeList(b *strings.Builder, name string, values []string) {
	values = uniqueNonEmpty(values)
	if len(values) == 0 {
		b.WriteString(name + ":\n- none\n\n")
		return
	}
	b.WriteString(name + ":\n")
	for _, value := range values {
		b.WriteString("- " + value + "\n")
	}
	b.WriteString("\n")
}

func writeTimeline(b *strings.Builder, operations []app.OperationResult) {
	if len(operations) == 0 {
		return
	}
	b.WriteString(FormatOperationTimeline(operations))
}

func FormatOperationTimeline(operations []app.OperationResult) string {
	var b strings.Builder
	b.WriteString("Workflow timeline:\n")
	for i, operation := range operations {
		id := valueOrDash(operation.ID)
		status := valueOrDash(string(operation.Status))
		message := valueOrDash(operation.Message)
		duration := operation.DurationMS
		if duration < 0 {
			duration = 0
		}
		line := fmt.Sprintf("%d. [%s] %s: %s in %d ms", i+1, status, id, message, duration)
		if operation.FailureCategory != "" && operation.Status == app.OperationStatusFailed {
			line += " (failure: " + operation.FailureCategory + ")"
		}
		if operation.ResultFile != "" {
			line += " -> " + operation.ResultFile
		} else if operation.Artifact != "" {
			line += " -> " + operation.Artifact
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	return safety.RedactString(b.String())
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
