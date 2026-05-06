package repair

import (
	"strings"

	"kigrepair/internal/cleaner"
	"kigrepair/internal/config"
	"kigrepair/internal/detector"
)

func PolicyDecisions(policy config.PolicySummary, detection detector.DetectionReport, preflight PreflightResult, defenderPlan DefenderPlanSummary, cleanupPlan cleaner.CleanupPlan) ([]string, []string) {
	decisions := []string{
		"Active policy profile: " + policy.Profile,
	}
	blocked := []string{}
	if !policy.AllowRealRepair {
		blocked = append(blocked, "repair_disabled_by_policy")
	}
	if policy.RequireRollbackSnapshot {
		decisions = append(decisions, "rollback_required")
	}
	if policy.RequireInstallerValidation {
		decisions = append(decisions, "installer_validation_required")
	}
	if policy.RequirePreflightReady && preflight.HasFailedRequiredCheck() {
		blocked = append(blocked, "preflight_not_ready")
	}
	if policy.StopOnDetectionUnknown && strings.EqualFold(string(detection.Health), string(detector.GrabberHealthUnknown)) {
		blocked = append(blocked, "unknown_detection_blocks_repair")
	}
	if policy.RequireDefenderBeforeInstall && policy.DefenderRequireCoverageForInstall && defenderPlan.Required && defenderPlan.ReadStatus == "unavailable" && !policy.DefenderTreatUnavailableAsWarning {
		blocked = append(blocked, "defender_required_but_unavailable")
	}
	if policy.RequireDefenderBeforeInstall && defenderPlan.Required && defenderPlan.WouldAddExclusion && !policy.DefenderAllowAddExclusion {
		blocked = append(blocked, "defender_exclusion_add_disabled_by_policy")
	}
	if policy.Profile == "conservative" && len(cleanupPlan.Skipped) > 0 {
		decisions = append(decisions, "Conservative profile will skip cleanup targets blocked by policy.")
	}
	if policy.Profile == "diagnostic" {
		decisions = append(decisions, "Diagnostic profile recommends collect-report before real repair.")
	}
	return decisions, blocked
}
