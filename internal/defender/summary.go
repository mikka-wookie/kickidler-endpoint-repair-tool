package defender

import (
	"fmt"
	"strings"

	"kigrepair/internal/detector"
)

func FormatPreflight(initial detector.DetectionReport, result DefenderEnsureResult, reportDir string) string {
	var b strings.Builder
	b.WriteString("Kigrepair Defender Ensure\n\n")
	b.WriteString("Install mode: " + string(initial.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(initial.InstallRoot) + "\n\n")
	b.WriteString("Required Defender paths:\n")
	for _, path := range result.RequiredPaths {
		b.WriteString("- " + path + "\n")
	}
	b.WriteString("\nMissing Defender exclusions:\n")
	for _, path := range result.MissingBefore {
		b.WriteString("- " + path + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func FormatSummary(result DefenderEnsureResult, final *detector.DetectionReport) string {
	var b strings.Builder
	if result.Ensure {
		b.WriteString("Kigrepair Defender Ensure\n\n")
	} else {
		b.WriteString("Kigrepair Defender Status\n\n")
	}
	b.WriteString("Install mode: " + valueOrDash(result.InstallMode) + "\n")
	b.WriteString("Install root: " + valueOrDash(result.InstallRoot) + "\n\n")
	if len(result.RequiredPaths) > 0 {
		b.WriteString("Required Defender paths:\n")
		for _, path := range result.RequiredPaths {
			b.WriteString("- " + path + "\n")
		}
		b.WriteString("\n")
	}
	if !result.Ensure {
		b.WriteString("Existing coverage:\n")
		for _, path := range result.RequiredPaths {
			status := "missing"
			for _, covered := range result.CoveredPaths {
				if strings.EqualFold(detector.NormalizeWindowsPath(path), detector.NormalizeWindowsPath(covered)) {
					status = "present"
					break
				}
			}
			b.WriteString(fmt.Sprintf("- %s: %s\n", path, status))
		}
		b.WriteString("\nNo changes were made.\n\n")
	} else if len(result.AddedPaths) > 0 || len(result.FailedPaths) > 0 {
		b.WriteString("Result:\n")
		for _, path := range result.AddedPaths {
			b.WriteString("- " + path + ": added\n")
		}
		for _, path := range result.FailedPaths {
			b.WriteString("- " + path + ": failed\n")
		}
		b.WriteString("\n")
	} else if len(result.MissingBefore) == 0 && len(result.RequiredPaths) > 0 {
		b.WriteString("All required Defender exclusions are already present.\n\n")
		b.WriteString("No changes were required.\n\n")
	}
	if result.Ensure && final != nil {
		b.WriteString("Final status:\n")
		if len(result.MissingAfter) == 0 && len(result.FailedPaths) == 0 {
			b.WriteString("- all required paths covered\n\n")
		} else {
			for _, path := range result.MissingAfter {
				b.WriteString("- " + path + ": missing\n")
			}
			b.WriteString("\n")
		}
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
	b.WriteString("Report:\n")
	b.WriteString(result.ReportDir)
	b.WriteString("\n")
	return b.String()
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
