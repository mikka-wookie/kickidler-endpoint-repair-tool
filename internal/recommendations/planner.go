package recommendations

import (
	"strings"

	"kigrepair/internal/detector"
)

const (
	issueHealthy                   = "healthy"
	issueNotInstalled              = "not_installed"
	issueServiceBinaryMissing      = "service_binary_missing"
	issueServiceNotRunning         = "service_not_running"
	issueWMIHiddenModeInconsistent = "wmi_hidden_mode_inconsistent"
	issueVerificationFailed        = "verification_failed"
	issueVerificationWarning       = "verification_warning"
	issueDefenderExclusionMissing  = "defender_exclusion_missing"
	issueDefenderStatusUnavailable = "defender_status_unavailable"
	issuePartialMSILeftovers       = "partial_msi_leftovers"
	issuePartialFilesLeftover      = "partial_files_leftover"
	issueUnknownInstallState       = "unknown_install_state"
)

func Plan(input RecommendationInput) RecommendationResult {
	primaryIssue, issues := issuesFromInput(input)
	actions := actionsForIssues(input, primaryIssue, issues)
	result := RecommendationResult{
		Status:  StatusInconclusive,
		Actions: actions,
	}
	if len(actions) > 0 {
		result.PrimaryAction = &result.Actions[0]
	}
	result.RequiredInputs = missingInputs(input, result.PrimaryAction)
	result.Warnings = warnings(input, result.PrimaryAction)
	result.Escalation = escalation(input, primaryIssue, result.PrimaryAction)
	result.Status = status(result)
	result.SupportSummary = SupportSummary(result)
	return result
}

func issuesFromInput(input RecommendationInput) (string, map[string]bool) {
	issues := map[string]bool{}
	if input.Classification != nil {
		for _, issue := range input.Classification.Issues {
			if issue.Code != "" {
				issues[issue.Code] = true
			}
		}
		if input.Classification.PrimaryIssue != nil && input.Classification.PrimaryIssue.Code != "" {
			issues[input.Classification.PrimaryIssue.Code] = true
			return input.Classification.PrimaryIssue.Code, issues
		}
		if strings.EqualFold(input.Classification.Status, "inconclusive") {
			issues[issueUnknownInstallState] = true
			return issueUnknownInstallState, issues
		}
	}
	if input.Detection != nil && input.Detection.Health == detector.GrabberHealthHealthy && !input.RepairFailed {
		if len(input.Detection.MissingDefenderPaths) > 0 {
			issues[issueDefenderExclusionMissing] = true
			return issueDefenderExclusionMissing, issues
		}
		issues[issueHealthy] = true
		return issueHealthy, issues
	}
	if input.RepairFailed || failedVerification(input.Verification) {
		issues[issueVerificationFailed] = true
		return issueVerificationFailed, issues
	}
	if warningVerification(input.Verification) {
		issues[issueVerificationWarning] = true
		return issueVerificationWarning, issues
	}
	if input.Detection == nil {
		issues[issueUnknownInstallState] = true
		return issueUnknownInstallState, issues
	}
	report := *input.Detection
	if !report.Defender.Available {
		issues[issueDefenderStatusUnavailable] = true
	}
	if len(report.MissingDefenderPaths) > 0 {
		issues[issueDefenderExclusionMissing] = true
	}
	switch report.Health {
	case detector.GrabberHealthNotInstalled:
		issues[issueNotInstalled] = true
		return issueNotInstalled, issues
	case detector.GrabberHealthPartiallyRemoved:
		if hasMSILeftovers(report) {
			issues[issuePartialMSILeftovers] = true
			return issuePartialMSILeftovers, issues
		}
		issues[issuePartialFilesLeftover] = true
		return issuePartialFilesLeftover, issues
	case detector.GrabberHealthBroken:
		if report.InstallMode == detector.InstallModeHiddenWMI && !report.ServiceExecutableExists {
			issues[issueWMIHiddenModeInconsistent] = true
			return issueWMIHiddenModeInconsistent, issues
		}
		if report.ServiceExecutablePath != "" && !report.ServiceExecutableExists {
			issues[issueServiceBinaryMissing] = true
			return issueServiceBinaryMissing, issues
		}
		if serviceNotRunning(report) {
			issues[issueServiceNotRunning] = true
			return issueServiceNotRunning, issues
		}
		issues[issueVerificationFailed] = true
		return issueVerificationFailed, issues
	default:
		issues[issueUnknownInstallState] = true
		return issueUnknownInstallState, issues
	}
}

func actionsForIssues(input RecommendationInput, primaryIssue string, issues map[string]bool) []RecommendedAction {
	switch primaryIssue {
	case issueHealthy:
		return []RecommendedAction{noRepairRequired()}
	case issueNotInstalled:
		return []RecommendedAction{installOrRepair(input)}
	case issueServiceBinaryMissing, issueServiceNotRunning, issueWMIHiddenModeInconsistent:
		return []RecommendedAction{fullRepair(input)}
	case issueVerificationFailed:
		return []RecommendedAction{escalateWithBundle(), collectSupportBundle()}
	case issuePartialMSILeftovers, issuePartialFilesLeftover:
		return []RecommendedAction{cleanupDryRun(), cleanupThenRepair(input)}
	case issueUnknownInstallState, issueVerificationWarning, issueDefenderStatusUnavailable:
		return []RecommendedAction{collectSupportBundle()}
	}
	if issues[issueDefenderExclusionMissing] {
		return []RecommendedAction{defenderEnsure()}
	}
	if len(issues) == 0 {
		return []RecommendedAction{collectSupportBundle()}
	}
	return []RecommendedAction{collectSupportBundle()}
}

func noRepairRequired() RecommendedAction {
	return RecommendedAction{
		Code:        ActionNoRepairRequired,
		Priority:    10,
		Title:       "No repair required",
		Description: "Grabber appears healthy. No repair is required.",
		Reason:      "Detection and verification do not show a repair condition.",
	}
}

func installOrRepair(input RecommendationInput) RecommendedAction {
	return RecommendedAction{
		Code:           ActionInstallOrRepairWithInvite,
		Priority:       30,
		Title:          "Install or repair with invite",
		Description:    "Grabber is not installed. Run repair with a supported MSI installer and Kickidler invite.",
		Command:        RepairCommand(input.InstallerPath),
		RequiresAdmin:  true,
		Destructive:    true,
		RequiresYes:    true,
		RequiredInputs: []string{InputInstaller, InputInvite},
		Reason:         "No Grabber services, files, or registry keys were detected.",
	}
}

func fullRepair(input RecommendationInput) RecommendedAction {
	return RecommendedAction{
		Code:           ActionRunFullRepair,
		Priority:       20,
		Title:          "Run full repair",
		Description:    "Run cleanup, Defender exclusion ensure, MSI reinstall, and final verification.",
		Command:        RepairCommand(input.InstallerPath),
		RequiresAdmin:  true,
		Destructive:    true,
		RequiresYes:    true,
		RequiredInputs: []string{InputInstaller, InputInvite},
		Reason:         "Current state indicates a broken Grabber installation that should be repaired end to end.",
	}
}

func defenderEnsure() RecommendedAction {
	return RecommendedAction{
		Code:          ActionRunDefenderEnsure,
		Priority:      50,
		Title:         "Ensure Defender exclusion",
		Description:   "Add the missing Windows Defender exclusion for the detected Grabber install root.",
		Command:       DefenderEnsureCommand(),
		RequiresAdmin: true,
		Destructive:   false,
		RequiresYes:   true,
		Reason:        "Defender exclusion coverage is missing and no stronger repair action was selected.",
	}
}

func cleanupDryRun() RecommendedAction {
	return RecommendedAction{
		Code:        ActionRunCleanupDryRun,
		Priority:    40,
		Title:       "Run cleanup dry-run",
		Description: "Preview cleanup targets before deleting files, services, registry keys, or MSI leftovers.",
		Command:     CleanupDryRunCommand(),
		Reason:      "Leftovers were detected without a working service. Support should inspect the cleanup plan first.",
	}
}

func cleanupThenRepair(input RecommendationInput) RecommendedAction {
	return RecommendedAction{
		Code:           ActionRunCleanupThenRepair,
		Priority:       45,
		Title:          "Cleanup then repair",
		Description:    "After reviewing the dry-run plan, clean leftovers and run full repair.",
		Command:        CleanupThenRepairCommand(input.InstallerPath),
		RequiresAdmin:  true,
		Destructive:    true,
		RequiresYes:    true,
		RequiredInputs: []string{InputInstaller, InputInvite},
		Reason:         "Partial leftovers usually need cleanup before reinstalling Grabber.",
	}
}

func collectSupportBundle() RecommendedAction {
	return RecommendedAction{
		Code:        ActionCollectSupportBundle,
		Priority:    60,
		Title:       "Collect support bundle",
		Description: "Collect read-only diagnostics and report archive for review.",
		Command:     CollectReportCommand(),
		Reason:      "The current state needs more diagnostics before a destructive repair recommendation.",
	}
}

func escalateWithBundle() RecommendedAction {
	return RecommendedAction{
		Code:        ActionEscalateWithBundle,
		Priority:    70,
		Title:       "Escalate with support bundle",
		Description: "Collect support bundle and escalate with the report archive.",
		Command:     CollectReportCommand(),
		Reason:      "Repair or final verification failed; escalation should include the latest report archive.",
	}
}

func missingInputs(input RecommendationInput, action *RecommendedAction) []RequiredInput {
	if action == nil {
		return nil
	}
	var result []RequiredInput
	for _, required := range action.RequiredInputs {
		switch required {
		case InputInstaller:
			if !input.HasInstaller && strings.TrimSpace(input.InstallerPath) == "" {
				result = append(result, RequiredInput{Name: InputInstaller, Description: "Path to supported Grabber MSI installer.", Secret: false})
			}
		case InputInvite:
			if !input.HasInvite {
				result = append(result, RequiredInput{Name: InputInvite, Description: "Kickidler invite value used by MSI installation.", Secret: true})
			}
		}
	}
	return result
}

func warnings(input RecommendationInput, action *RecommendedAction) []string {
	if action == nil {
		return nil
	}
	var result []string
	if action.RequiresAdmin && !input.IsAdmin {
		result = append(result, "Recommended action requires administrator rights.")
	}
	return result
}

func escalation(input RecommendationInput, primaryIssue string, action *RecommendedAction) *EscalationAdvice {
	if action == nil {
		return nil
	}
	if action.Code != ActionEscalateWithBundle && primaryIssue != issueUnknownInstallState {
		return nil
	}
	reason := "State is unknown or inconclusive."
	if input.RepairFailed || failedVerification(input.Verification) {
		reason = "Repair or final verification failed."
	}
	return &EscalationAdvice{
		Needed:       true,
		Reason:       reason,
		IncludeFiles: []string{"summary.txt", "initial-detection.json", "final-detection.json", "verification-result.json", "repair-result.json", "operations.json", "repair.log"},
	}
}

func status(result RecommendationResult) string {
	if result.PrimaryAction == nil {
		return StatusInconclusive
	}
	if len(result.RequiredInputs) > 0 {
		return StatusMissingRequiredInput
	}
	if result.Escalation != nil && result.Escalation.Needed {
		return StatusEscalationRecommended
	}
	if result.PrimaryAction.Code == ActionNoRepairRequired {
		return StatusNoActionRequired
	}
	return StatusActionRecommended
}

func failedVerification(result *VerificationState) bool {
	return result != nil && strings.EqualFold(result.OverallStatus, "failed")
}

func warningVerification(result *VerificationState) bool {
	return result != nil && strings.EqualFold(result.OverallStatus, "warning")
}

func serviceNotRunning(report detector.DetectionReport) bool {
	for _, service := range report.Services {
		if strings.EqualFold(service.Name, report.PrimaryService) && service.Exists {
			return !strings.EqualFold(service.Status, "running")
		}
	}
	return false
}

func hasMSILeftovers(report detector.DetectionReport) bool {
	for _, key := range report.Registry {
		path := strings.ToLower(key.Path)
		if key.Exists && (strings.Contains(path, `installer\`) || strings.Contains(path, `uninstall\{eb1fbc37`)) {
			return true
		}
	}
	return false
}
