package repair

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/detector"
	"kigrepair/internal/recommendations"
	"kigrepair/internal/reports"
	"kigrepair/internal/verifier"
)

func FormatPreflight(report detector.DetectionReport, decision Decision, reportDir string) string {
	var b strings.Builder
	b.WriteString("Kigrepair Repair\n\n")
	b.WriteString("Initial health: " + string(report.Health) + "\n")
	b.WriteString("Initial install mode: " + string(report.InstallMode) + "\n")
	b.WriteString("Initial install root: " + valueOrDash(report.InstallRoot) + "\n\n")
	b.WriteString("Planned workflow:\n")
	b.WriteString("- Cleanup old/broken installation: " + yesNo(decision.CleanupNeeded) + "\n")
	b.WriteString("- Install MSI: " + yesNo(decision.InstallNeeded) + "\n")
	b.WriteString("- Ensure Defender exclusion: " + yesNo(decision.DefenderNeeded) + "\n")
	b.WriteString("- Final verification: yes\n\n")
	return b.String()
}

func finishedOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func FormatSummary(result RepairResult, final *detector.DetectionReport) string {
	var b strings.Builder
	b.WriteString("Kigrepair Repair\n\n")
	b.WriteString("Initial health: " + valueOrDash(result.InitialHealth) + "\n")
	b.WriteString("Install mode: " + valueOrDash(result.InitialInstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(result.InitialInstallRoot) + "\n\n")
	if result.InstallExecuted {
		b.WriteString("Selected installer: " + valueOrDash(result.InstallerPath) + "\n")
		if result.InstallerResolution != nil && result.InstallerResolution.SelectedSource != "" {
			b.WriteString("Installer source: " + string(result.InstallerResolution.SelectedSource) + "\n")
		}
		b.WriteString("\n")
	} else {
		b.WriteString("Installer: not required\n\n")
	}
	if !result.CleanupExecuted && !result.InstallExecuted && result.InitialHealth == string(detector.GrabberHealthHealthy) {
		b.WriteString("No reinstall required.\n")
		if final != nil {
			b.WriteString("Defender exclusion: " + defenderExclusionStatus(*final) + "\n\n")
		}
	} else {
		b.WriteString("Execution:\n")
		b.WriteString("- Cleanup: " + executedStatus(result.CleanupExecuted, result.CleanupNeeded) + "\n")
		b.WriteString("- Install: " + executedStatus(result.InstallExecuted, result.InstallExecuted) + "\n")
		b.WriteString("- Defender exclusion: " + executedStatus(result.DefenderExecuted, result.DefenderExecuted) + "\n")
		b.WriteString("- Final verification: " + finalVerificationStatus(result.ExitCode) + "\n\n")
	}
	if final != nil {
		b.WriteString("Final health: " + string(final.Health) + "\n")
		b.WriteString("Install mode: " + string(final.InstallMode) + "\n")
		b.WriteString("Install root: " + valueOrDash(final.InstallRoot) + "\n")
		b.WriteString("Primary service: " + valueOrDash(final.PrimaryService) + "\n")
		b.WriteString("Service status: " + serviceStatus(*final) + "\n")
		b.WriteString("Service executable: " + valueOrDash(final.ServiceExecutablePath) + "\n")
		b.WriteString("Defender exclusion: " + defenderExclusionStatus(*final) + "\n")
	} else if result.FinalHealth != "" {
		b.WriteString("Final health: " + result.FinalHealth + "\n")
	}
	if result.RebootRequired {
		b.WriteString("Reboot required: yes\n")
	}
	b.WriteString(fmt.Sprintf("Exit code: %d\n\n", result.ExitCode))
	if result.Verification != nil {
		verificationRecommendation := recommendations.RecommendationResult{}
		if result.Recommendation != nil {
			verificationRecommendation = *result.Recommendation
		}
		b.WriteString(verifier.FormatSummary(*result.Verification, verificationRecommendation))
		b.WriteString("\n")
	}
	if result.Verification == nil && result.Recommendation != nil {
		b.WriteString(recommendations.FormatSection(*result.Recommendation))
	}
	if len(result.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range unique(result.Warnings) {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}
	if len(result.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, errText := range unique(result.Errors) {
			b.WriteString("- " + errText + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Report:\n")
	b.WriteString(result.ReportDir)
	b.WriteString("\n")
	finalHealth := result.FinalHealth
	installMode := result.FinalInstallMode
	installRoot := result.FinalInstallRoot
	primaryService := ""
	if final != nil {
		finalHealth = string(final.Health)
		installMode = string(final.InstallMode)
		installRoot = final.InstallRoot
		primaryService = final.PrimaryService
	}
	return reports.FormatSummary(reports.SummaryData{
		Command:        "repair",
		Started:        result.StartedAt,
		Finished:       finishedOrNow(result.FinishedAt),
		Mode:           result.Mode,
		ExitCode:       result.ExitCode,
		ReportDir:      result.ReportDir,
		InitialHealth:  result.InitialHealth,
		FinalHealth:    finalHealth,
		InstallMode:    installMode,
		InstallRoot:    installRoot,
		PrimaryService: primaryService,
		Warnings:       result.Warnings,
		Errors:         result.Errors,
		Actions: []string{
			"Cleanup: " + executedStatus(result.CleanupExecuted, result.CleanupNeeded),
			"Install: " + executedStatus(result.InstallExecuted, result.InstallExecuted),
			"Defender exclusion: " + executedStatus(result.DefenderExecuted, result.DefenderExecuted),
			"Final verification: " + finalVerificationStatus(result.ExitCode),
		},
	}, b.String())
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func executedStatus(executed bool, expected bool) string {
	if executed {
		return "success"
	}
	if expected {
		return "not executed"
	}
	return "skipped"
}

func finalVerificationStatus(exitCode int) string {
	switch exitCode {
	case ExitVerificationFailed:
		return "failed"
	case ExitWarnings, ExitRebootRequired:
		return "warning"
	default:
		return "success"
	}
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

func unique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
