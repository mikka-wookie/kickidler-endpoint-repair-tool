package reports

import (
	"fmt"
	"strings"
	"time"
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
}

func FormatSummary(data SummaryData, details string) string {
	finished := data.Finished
	if finished.IsZero() {
		finished = time.Now()
	}
	var b strings.Builder
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
	if strings.TrimSpace(details) != "" {
		b.WriteString(details)
		if !strings.HasSuffix(details, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
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
