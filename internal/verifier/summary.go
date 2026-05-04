package verifier

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/recommendations"
	"kigrepair/internal/reports"
)

func FormatSummary(result VerificationResult, recommendationsInput ...recommendations.RecommendationResult) string {
	recommendation, hasRecommendation := optionalRecommendation(recommendationsInput)
	var b strings.Builder
	b.WriteString("Kigrepair Verify\n\n")
	b.WriteString("Verification: " + string(result.OverallStatus) + "\n")
	b.WriteString("Health: " + valueOrDash(result.Health) + "\n")
	b.WriteString("Install mode: " + valueOrDash(result.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(result.InstallRoot) + "\n")
	b.WriteString("Primary service: " + valueOrDash(result.PrimaryService) + "\n")
	b.WriteString("Service status: " + valueOrDash(result.ServiceStatus) + "\n")
	b.WriteString("Service executable: " + valueOrDash(result.ServiceExecutable) + "\n")
	if result.ProcessStatus != "" {
		b.WriteString("Expected process running: " + processWord(result.ProcessStatus) + "\n")
	}
	if result.DefenderStatus != "" {
		b.WriteString("Defender exclusion: " + result.DefenderStatus + "\n")
	}
	if result.MSIInstallLogState != "" {
		b.WriteString("MSI install log: " + result.MSIInstallLogState + "\n")
	}
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
	if hasRecommendation {
		b.WriteString(recommendations.FormatConsole(recommendation))
		b.WriteString("\n")
	} else {
		b.WriteString("Next recommended support action:\n")
		b.WriteString("- " + nextAction(result) + "\n\n")
	}
	if result.ReportDir != "" {
		b.WriteString("Report:\n")
		b.WriteString(result.ReportDir)
		b.WriteString("\n")
	}

	details := b.String()
	if hasRecommendation {
		details += recommendations.FormatSection(recommendation)
	}
	return reports.FormatSummary(reports.SummaryData{
		Command:        commandOrVerify(result.Command),
		Started:        result.StartedAt,
		Finished:       finishedOrNow(result.FinishedAt),
		Mode:           result.Mode,
		ExitCode:       result.ExitCode,
		ReportDir:      result.ReportDir,
		FinalHealth:    result.Health,
		InstallMode:    result.InstallMode,
		InstallRoot:    result.InstallRoot,
		PrimaryService: result.PrimaryService,
		Warnings:       result.Warnings,
		Errors:         result.Errors,
		Actions:        []string{"Verification: " + string(result.OverallStatus)},
	}, details)
}

func FormatConsoleSummary(result VerificationResult, recommendationsInput ...recommendations.RecommendationResult) string {
	recommendation, hasRecommendation := optionalRecommendation(recommendationsInput)
	var b strings.Builder
	b.WriteString("Kigrepair Verify\n\n")
	b.WriteString("Report directory: " + valueOrDash(result.ReportDir) + "\n")
	b.WriteString("Detected health: " + valueOrDash(result.Health) + "\n")
	b.WriteString("Install mode: " + valueOrDash(result.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(result.InstallRoot) + "\n")
	b.WriteString("Verification status: " + string(result.OverallStatus) + "\n\n")
	writeNonSuccessChecks(&b, result)
	if hasRecommendation {
		b.WriteString(recommendations.FormatConsole(recommendation))
	}
	return b.String()
}

func optionalRecommendation(values []recommendations.RecommendationResult) (recommendations.RecommendationResult, bool) {
	if len(values) == 0 {
		return recommendations.RecommendationResult{}, false
	}
	return values[0], true
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func finishedOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func commandOrVerify(command string) string {
	if strings.TrimSpace(command) == "" {
		return "verify"
	}
	return command
}

func processWord(status string) string {
	if status == "found" {
		return "yes"
	}
	return "not detected"
}

func nextAction(result VerificationResult) string {
	if result.OverallStatus == VerificationPassed {
		return "No repair required."
	}
	if result.OverallStatus == VerificationFailed {
		return "Run repair with a valid installer and invite."
	}
	return "Review detection and verification JSON files."
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

func writeNonSuccessChecks(b *strings.Builder, result VerificationResult) {
	wrote := false
	for _, check := range result.Checks {
		if check.Status != VerificationFailed && check.Status != VerificationWarning {
			continue
		}
		if !wrote {
			b.WriteString("Failed/warning checks:\n")
			wrote = true
		}
		b.WriteString("- " + check.Name + ": " + string(check.Status))
		if strings.TrimSpace(check.Message) != "" {
			b.WriteString(" (" + check.Message + ")")
		}
		b.WriteString("\n")
	}
	if wrote {
		b.WriteString("\n")
		return
	}
	b.WriteString("Failed/warning checks:\n- none\n\n")
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
