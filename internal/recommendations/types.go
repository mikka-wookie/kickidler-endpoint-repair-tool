package recommendations

import (
	"kigrepair/internal/detector"
)

const (
	StatusNoActionRequired      = "no_action_required"
	StatusActionRecommended     = "action_recommended"
	StatusMissingRequiredInput  = "missing_required_input"
	StatusEscalationRecommended = "escalation_recommended"
	StatusInconclusive          = "inconclusive"
)

const (
	ActionNoRepairRequired          = "no_repair_required"
	ActionInstallOrRepairWithInvite = "install_or_repair_with_invite"
	ActionRunFullRepair             = "run_full_repair"
	ActionRunDefenderEnsure         = "run_defender_ensure"
	ActionRunCleanupDryRun          = "run_cleanup_dry_run"
	ActionRunCleanupThenRepair      = "run_cleanup_then_repair"
	ActionCollectSupportBundle      = "collect_support_bundle"
	ActionEscalateWithBundle        = "escalate_with_bundle"
)

const (
	InputInstaller = "installer"
	InputInvite    = "invite"
)

type RecommendationInput struct {
	Detection      *detector.DetectionReport
	Verification   *VerificationState
	Classification *ClassificationResult

	InstallerPath string
	HasInstaller  bool
	HasInvite     bool
	IsAdmin       bool
	IsInteractive bool
	OutputDir     string

	RepairFailed bool
}

type VerificationState struct {
	OverallStatus string   `json:"overall_status"`
	Warnings      []string `json:"warnings,omitempty"`
	Errors        []string `json:"errors,omitempty"`
}

type ClassificationResult struct {
	Status       string            `json:"status"`
	PrimaryIssue *ClassifiedIssue  `json:"primary_issue,omitempty"`
	Issues       []ClassifiedIssue `json:"issues,omitempty"`
}

type ClassifiedIssue struct {
	Code        string `json:"code"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Severity    string `json:"severity,omitempty"`
}

type RecommendationResult struct {
	Status         string              `json:"status"`
	PrimaryAction  *RecommendedAction  `json:"primary_action,omitempty"`
	Actions        []RecommendedAction `json:"actions"`
	RequiredInputs []RequiredInput     `json:"required_inputs,omitempty"`
	Warnings       []string            `json:"warnings,omitempty"`
	Escalation     *EscalationAdvice   `json:"escalation,omitempty"`
	SupportSummary string              `json:"support_summary"`
}

type RecommendedAction struct {
	Code           string   `json:"code"`
	Priority       int      `json:"priority"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Command        string   `json:"command,omitempty"`
	RequiresAdmin  bool     `json:"requires_admin"`
	Destructive    bool     `json:"destructive"`
	RequiresYes    bool     `json:"requires_yes"`
	RequiredInputs []string `json:"required_inputs,omitempty"`
	Reason         string   `json:"reason"`
}

type RequiredInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Secret      bool   `json:"secret"`
}

type EscalationAdvice struct {
	Needed       bool     `json:"needed"`
	Reason       string   `json:"reason,omitempty"`
	IncludeFiles []string `json:"include_files,omitempty"`
}
