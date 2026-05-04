package verifier

import (
	"time"

	"kigrepair/internal/detector"
)

type VerificationStatus string

const (
	VerificationSuccess VerificationStatus = "success"
	VerificationWarning VerificationStatus = "warning"
	VerificationFailed  VerificationStatus = "failed"
)

type CheckStatus string

const (
	CheckSuccess CheckStatus = "success"
	CheckFailed  CheckStatus = "failed"
	CheckSkipped CheckStatus = "skipped"
	CheckWarning CheckStatus = "warning"
)

type VerificationCheck struct {
	Name      string      `json:"name"`
	Target    string      `json:"target,omitempty"`
	Status    CheckStatus `json:"status"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

type Options struct {
	InstallExecuted bool
	MSIInstallLog   string
}

type VerificationResult struct {
	Command         string                   `json:"command,omitempty"`
	StartedAt       time.Time                `json:"started_at,omitempty"`
	FinishedAt      time.Time                `json:"finished_at,omitempty"`
	Mode            string                   `json:"mode,omitempty"`
	ReportDir       string                   `json:"report_dir,omitempty"`
	ExitCode        int                      `json:"exit_code"`
	Status          VerificationStatus       `json:"status"`
	Message         string                   `json:"message"`
	Health          string                   `json:"health,omitempty"`
	InstallMode     string                   `json:"install_mode,omitempty"`
	InstallRoot     string                   `json:"install_root,omitempty"`
	PrimaryService  string                   `json:"primary_service,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
	Errors          []string                 `json:"errors,omitempty"`
	Checks          []VerificationCheck      `json:"checks"`
	Detection       detector.DetectionReport `json:"detection"`
	InstallExecuted bool                     `json:"install_executed"`
	MSIInstallLog   string                   `json:"msi_install_log,omitempty"`
}
