package verifier

import (
	"time"

	"kigrepair/internal/detector"
)

type VerificationStatus string

const (
	VerificationPassed  VerificationStatus = "passed"
	VerificationWarning VerificationStatus = "warning"
	VerificationFailed  VerificationStatus = "failed"
	VerificationSkipped VerificationStatus = "skipped"
)

type VerificationCheck struct {
	Name    string             `json:"name"`
	Target  string             `json:"target,omitempty"`
	Status  VerificationStatus `json:"status"`
	Message string             `json:"message,omitempty"`
	Error   string             `json:"error,omitempty"`
}

type VerificationResult struct {
	Command            string                   `json:"command,omitempty"`
	StartedAt          time.Time                `json:"started_at"`
	FinishedAt         time.Time                `json:"finished_at"`
	Mode               string                   `json:"mode,omitempty"`
	ReportDir          string                   `json:"report_dir,omitempty"`
	ExitCode           int                      `json:"exit_code"`
	OverallStatus      VerificationStatus       `json:"overall_status"`
	Health             string                   `json:"health"`
	InstallMode        string                   `json:"install_mode"`
	InstallRoot        string                   `json:"install_root"`
	PrimaryService     string                   `json:"primary_service"`
	ServiceStatus      string                   `json:"service_status"`
	ServiceExecutable  string                   `json:"service_executable"`
	Checks             []VerificationCheck      `json:"checks"`
	Warnings           []string                 `json:"warnings,omitempty"`
	Errors             []string                 `json:"errors,omitempty"`
	ProcessStatus      string                   `json:"process_status,omitempty"`
	DefenderStatus     string                   `json:"defender_status,omitempty"`
	MSIInstallLogState string                   `json:"msi_install_log_state,omitempty"`
	Detection          detector.DetectionReport `json:"detection,omitempty"`
}

type VerifyOptions struct {
	InstallExecuted            bool
	MSIInstallLogPath          string
	RequireRunningProcess      bool
	AllowDefenderUnavailable   bool
	ExpectInstalledState       bool
	AllowStoppedServiceWarning bool
}
