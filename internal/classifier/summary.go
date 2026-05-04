package classifier

import (
	"strings"

	"kigrepair/internal/detector"
)

func SupportSummary(input ClassificationInput, result ClassificationResult) string {
	if result.PrimaryIssue == nil {
		return "Grabber installation state is inconclusive. Detection returned conflicting or insufficient signals. Review detection JSON and collect a support bundle for escalation."
	}
	d := input.Detection
	switch result.PrimaryIssue.Code {
	case CodeHealthy:
		return healthySummary(d)
	case CodeServiceBinaryMissing:
		return serviceBinaryMissingSummary(d, hasIssue(result.Issues, CodeDefenderExclusionMissing))
	case CodeNotInstalled:
		return "Grabber is not installed. No supported Grabber service, known install root, or product registry presence was detected. Recommended action: install Grabber or run repair if the support case requires restoration."
	case CodeUnknownInstallState:
		return "Grabber installation state is inconclusive. Detection returned conflicting or insufficient signals. Review detection JSON and collect a support bundle for escalation."
	default:
		return result.PrimaryIssue.Description + " Recommended action: " + result.PrimaryIssue.Action
	}
}

func FormatSection(result ClassificationResult) string {
	var b strings.Builder
	b.WriteString("Classification\n")
	b.WriteString("--------------\n")
	b.WriteString("Status: " + valueOrDash(result.Status) + "\n")
	if result.PrimaryIssue != nil {
		b.WriteString("Primary issue: " + result.PrimaryIssue.Code + "\n")
		b.WriteString("Severity: " + result.PrimaryIssue.Severity + "\n")
	} else {
		b.WriteString("Primary issue: -\n")
		b.WriteString("Severity: -\n")
	}
	b.WriteString("Summary: " + valueOrDash(result.SupportSummary) + "\n")
	b.WriteString("Recommended action: " + valueOrDash(result.RecommendedAction) + "\n")
	return b.String()
}

func FormatConsole(result ClassificationResult) string {
	var b strings.Builder
	if result.PrimaryIssue != nil {
		b.WriteString("Classification: " + result.PrimaryIssue.Code + "\n")
		b.WriteString("Severity: " + result.PrimaryIssue.Severity + "\n")
	} else {
		b.WriteString("Classification: -\n")
		b.WriteString("Severity: -\n")
	}
	b.WriteString("Recommended action: " + valueOrDash(result.RecommendedAction) + "\n")
	return b.String()
}

func healthySummary(d *detector.DetectionReport) string {
	service := "-"
	mode := "-"
	if d != nil {
		service = valueOrDash(d.PrimaryService)
		mode = valueOrDash(string(d.InstallMode))
	}
	return "Grabber installation appears healthy. Service " + service + " is present and running, executable exists, install mode is " + mode + ", and Defender exclusion coverage is valid. No repair is required."
}

func serviceBinaryMissingSummary(d *detector.DetectionReport, defenderMissing bool) string {
	service := "-"
	exe := "-"
	if d != nil {
		service = valueOrDash(d.PrimaryService)
		exe = valueOrDash(d.ServiceExecutablePath)
	}
	summary := "Grabber installation is broken. Service " + service + " exists and points to " + exe + ", but the executable is missing."
	if defenderMissing {
		summary += " Defender exclusion coverage is also missing, so Microsoft Defender may have removed the binary."
	}
	return summary + " Recommended action: run repair with a valid installer and invite."
}
