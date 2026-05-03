package defender

import "time"

type DefenderEnsureResult struct {
	StartedAt          time.Time `json:"started_at"`
	FinishedAt         time.Time `json:"finished_at"`
	Ensure             bool      `json:"ensure"`
	AllKnownPaths      bool      `json:"all_known_paths,omitempty"`
	InstallMode        string    `json:"install_mode"`
	InstallRoot        string    `json:"install_root,omitempty"`
	RequiredPaths      []string  `json:"required_paths"`
	ExistingExclusions []string  `json:"existing_exclusions"`
	CoveredPaths       []string  `json:"covered_paths"`
	MissingBefore      []string  `json:"missing_before"`
	AddedPaths         []string  `json:"added_paths"`
	FailedPaths        []string  `json:"failed_paths"`
	MissingAfter       []string  `json:"missing_after"`
	Warnings           []string  `json:"warnings"`
	Errors             []string  `json:"errors"`
	ReportDir          string    `json:"report_dir"`
	ExitCode           int       `json:"exit_code"`
}

const (
	ExitOK                   = 0
	ExitWarnings             = 1
	ExitAdminRequired        = 2
	ExitConfirmationRequired = 3
	ExitNoInstallationRoot   = 4
	ExitAddFailed            = 5
	ExitVerificationFailed   = 6
	ExitUnexpectedError      = 10
)
