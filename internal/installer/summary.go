package installer

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/detector"
	"kigrepair/internal/reports"
	"kigrepair/internal/verifier"
)

func FormatInstallSummary(result InstallResult, final *detector.DetectionReport) string {
	var b strings.Builder
	b.WriteString("Kigrepair Install\n\n")
	if result.Resolution.SelectedSource == InstallerSourceExplicit {
		b.WriteString("Installer: explicit\n")
	} else if result.Resolution.SelectedPath != "" {
		b.WriteString("Installer: auto-detected\n")
	} else {
		b.WriteString("Installer: " + valueOrDash(result.InstallerPath) + "\n")
	}
	if result.Resolution.SelectedPath != "" {
		b.WriteString("Selected installer: " + result.Resolution.SelectedPath + "\n")
		b.WriteString("Installer source: " + string(result.Resolution.SelectedSource) + "\n")
	}
	if result.InviteProvided {
		b.WriteString("Invite: provided\n")
	} else {
		b.WriteString("Invite: missing\n")
	}
	b.WriteString("Initial health: " + valueOrDash(result.InitialHealth) + "\n\n")
	b.WriteString("MSI install: " + valueOrDash(result.MSI.Status) + "\n")
	b.WriteString(fmt.Sprintf("MSI exit code: %d\n\n", result.MSI.ExitCode))
	if final != nil {
		b.WriteString("Final health: " + string(final.Health) + "\n")
		b.WriteString("Install mode: " + string(final.InstallMode) + "\n")
		b.WriteString("Install root: " + valueOrDash(final.InstallRoot) + "\n")
		b.WriteString("Primary service: " + valueOrDash(final.PrimaryService) + "\n")
		b.WriteString("Service status: " + serviceStatus(*final) + "\n")
		b.WriteString("Service executable: " + valueOrDash(final.ServiceExecutablePath) + "\n")
		b.WriteString("Defender exclusion: " + defenderExclusionStatus(*final) + "\n\n")
	} else if result.FinalHealth != "" {
		b.WriteString("Final health: " + result.FinalHealth + "\n\n")
	}
	if result.ExitCode == ExitInstallWarnings || len(result.Warnings) > 0 {
		b.WriteString("Install completed with warning\n\n")
	}
	if result.Verification != nil {
		b.WriteString(verifierSummary(*result.Verification))
		b.WriteString("\n")
	}
	if len(result.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range result.Warnings {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}
	if len(result.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, errText := range result.Errors {
			b.WriteString("- " + errText + "\n")
		}
		b.WriteString("\n")
	}
	if hasMissingDefenderWarning(result.Warnings) {
		b.WriteString("Recommendations:\n")
		b.WriteString("- Run: kigrepair.exe defender --ensure\n")
		b.WriteString("- Or run full repair workflow when implemented\n\n")
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
		Command:        "install",
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
		Actions:        []string{"MSI install: " + valueOrDash(result.MSI.Status)},
	}, b.String())
}

func verifierSummary(result verifier.VerificationResult) string {
	return verifier.FormatSummary(result)
}

func finishedOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
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

func hasMissingDefenderWarning(warnings []string) bool {
	for _, warning := range warnings {
		if strings.HasPrefix(warning, "Missing Defender exclusion:") {
			return true
		}
	}
	return false
}
