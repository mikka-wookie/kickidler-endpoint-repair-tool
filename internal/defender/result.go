package defender

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/rollback"
)

type DefenderEnsureResult struct {
	StartedAt          time.Time         `json:"started_at"`
	FinishedAt         time.Time         `json:"finished_at"`
	Mode               string            `json:"mode"`
	Ensure             bool              `json:"ensure"`
	AllKnownPaths      bool              `json:"all_known_paths,omitempty"`
	InstallMode        string            `json:"install_mode"`
	InstallRoot        string            `json:"install_root,omitempty"`
	RequiredPaths      []string          `json:"required_paths"`
	ExistingExclusions []string          `json:"existing_exclusions"`
	CoveredPaths       []string          `json:"covered_paths"`
	MissingBefore      []string          `json:"missing_before"`
	AddedPaths         []string          `json:"added_paths"`
	FailedPaths        []string          `json:"failed_paths"`
	MissingAfter       []string          `json:"missing_after"`
	Rollback           *rollback.Summary `json:"rollback,omitempty"`
	Warnings           []string          `json:"warnings"`
	Errors             []string          `json:"errors"`
	ReportDir          string            `json:"report_dir"`
	ExitCode           int               `json:"exit_code"`
}

const (
	ExitOK                   = app.ExitSuccess
	ExitWarnings             = app.ExitWarnings
	ExitAdminRequired        = app.ExitAdminRequired
	ExitConfirmationRequired = app.ExitConfirmationRequired
	ExitNoInstallationRoot   = app.ExitInvalidInput
	ExitAddFailed            = app.ExitDefenderFailed
	ExitVerificationFailed   = app.ExitVerificationFailed
	ExitUnexpectedError      = app.ExitUnexpectedError
)
