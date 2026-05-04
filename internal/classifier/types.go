package classifier

import (
	"kigrepair/internal/detector"
)

const (
	StatusNoIssue       = "no_issue"
	StatusIssueDetected = "issue_detected"
	StatusInconclusive  = "inconclusive"

	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"

	CodeHealthy                   = "healthy"
	CodeNotInstalled              = "not_installed"
	CodeServiceBinaryMissing      = "service_binary_missing"
	CodeServiceNotRunning         = "service_not_running"
	CodeDefenderExclusionMissing  = "defender_exclusion_missing"
	CodePartialMSILeftovers       = "partial_msi_leftovers"
	CodePartialFilesLeftover      = "partial_files_leftover"
	CodeWMIHiddenModeInconsistent = "wmi_hidden_mode_inconsistent"
	CodeUnknownInstallState       = "unknown_install_state"
	CodeVerificationFailed        = "verification_failed"
	CodeVerificationWarning       = "verification_warning"
	CodeDefenderStatusUnavailable = "defender_status_unavailable"
)

type ClassificationInput struct {
	Detection    *detector.DetectionReport
	Verification any
}

type ClassificationResult struct {
	Status            string  `json:"status"`
	PrimaryIssue      *Issue  `json:"primary_issue,omitempty"`
	Issues            []Issue `json:"issues"`
	RecommendedAction string  `json:"recommended_action"`
	SupportSummary    string  `json:"support_summary"`
}

type Issue struct {
	Code        string   `json:"code"`
	Severity    string   `json:"severity"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Evidence    []string `json:"evidence,omitempty"`
	Action      string   `json:"action"`
}
