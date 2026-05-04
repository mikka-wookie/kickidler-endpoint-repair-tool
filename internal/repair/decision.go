package repair

import (
	"strings"

	"kigrepair/internal/detector"
)

type Decision struct {
	CleanupNeeded  bool     `json:"cleanup_needed"`
	InstallNeeded  bool     `json:"install_needed"`
	DefenderNeeded bool     `json:"defender_needed"`
	Stop           bool     `json:"stop"`
	Warnings       []string `json:"warnings,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

func Decide(report detector.DetectionReport, force bool, cleanupActionCount int) Decision {
	decision := Decision{}
	switch report.Health {
	case detector.GrabberHealthHealthy:
		decision.CleanupNeeded = force
		decision.InstallNeeded = force
		decision.DefenderNeeded = force || defenderMissing(report)
	case detector.GrabberHealthBroken, detector.GrabberHealthPartiallyRemoved:
		decision.CleanupNeeded = true
		decision.InstallNeeded = true
		decision.DefenderNeeded = true
	case detector.GrabberHealthNotInstalled:
		decision.CleanupNeeded = cleanupActionCount > 0
		decision.InstallNeeded = true
		decision.DefenderNeeded = true
	case detector.GrabberHealthUnknown:
		if !force {
			decision.Stop = true
			decision.Errors = append(decision.Errors, "Initial health is unknown; rerun repair with --force after reviewing the report")
			return decision
		}
		decision.CleanupNeeded = true
		decision.InstallNeeded = true
		decision.DefenderNeeded = true
	default:
		if !force {
			decision.Stop = true
			decision.Errors = append(decision.Errors, "Initial health could not be classified; rerun repair with --force after reviewing the report")
			return decision
		}
		decision.CleanupNeeded = true
		decision.InstallNeeded = true
		decision.DefenderNeeded = true
	}
	return decision
}

func defenderMissing(report detector.DetectionReport) bool {
	if strings.TrimSpace(report.InstallRoot) == "" || !report.Defender.Available {
		return false
	}
	return len(report.MissingDefenderPaths) > 0 || len(report.Defender.MissingPaths) > 0
}
