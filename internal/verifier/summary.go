package verifier

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/detector"
	"kigrepair/internal/reports"
)

func FormatSummary(result VerificationResult) string {
	report := result.Detection
	var b strings.Builder
	b.WriteString("Kigrepair Verify\n\n")
	b.WriteString("Verification: " + string(result.Status) + "\n")
	b.WriteString("Health: " + string(report.Health) + "\n")
	b.WriteString("Install mode: " + string(report.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(report.InstallRoot) + "\n")
	b.WriteString("Primary service: " + valueOrDash(report.PrimaryService) + "\n")
	b.WriteString("Service status: " + serviceStatus(report) + "\n")
	b.WriteString("Service executable: " + valueOrDash(report.ServiceExecutablePath) + "\n")
	b.WriteString("Defender exclusion: " + defenderExclusionStatus(report) + "\n")
	b.WriteString(fmt.Sprintf("Exit code: %d\n\n", result.ExitCode))
	b.WriteString("Checks:\n")
	for _, check := range result.Checks {
		b.WriteString("- " + check.Name + ": " + string(check.Status))
		if strings.TrimSpace(check.Message) != "" {
			b.WriteString(" (" + check.Message + ")")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	writeList(&b, "Warnings", result.Warnings)
	writeList(&b, "Errors", result.Errors)
	b.WriteString("Report:\n")
	b.WriteString(result.ReportDir)
	b.WriteString("\n")

	return reports.FormatSummary(reports.SummaryData{
		Command:        "verify",
		Started:        result.StartedAt,
		Finished:       finishedOrNow(result.FinishedAt),
		Mode:           result.Mode,
		ExitCode:       result.ExitCode,
		ReportDir:      result.ReportDir,
		FinalHealth:    string(report.Health),
		InstallMode:    string(report.InstallMode),
		InstallRoot:    report.InstallRoot,
		PrimaryService: report.PrimaryService,
		Warnings:       result.Warnings,
		Errors:         result.Errors,
		Actions:        []string{"Verification: " + string(result.Status)},
	}, b.String())
}

func finishedOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
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

func writeList(b *strings.Builder, name string, values []string) {
	if len(values) == 0 {
		b.WriteString(name + ":\n- none\n\n")
		return
	}
	b.WriteString(name + ":\n")
	for _, value := range unique(values) {
		b.WriteString("- " + value + "\n")
	}
	b.WriteString("\n")
}

func unique(values []string) []string {
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
