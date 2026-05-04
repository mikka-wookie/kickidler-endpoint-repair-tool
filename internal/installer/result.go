package installer

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/verifier"
)

type InstallResult struct {
	StartedAt        time.Time                    `json:"started_at"`
	FinishedAt       time.Time                    `json:"finished_at"`
	Mode             string                       `json:"mode"`
	InstallerPath    string                       `json:"installer_path"`
	Resolution       InstallerResolution          `json:"installer_resolution"`
	InviteProvided   bool                         `json:"invite_provided"`
	MSI              MSIResult                    `json:"msi"`
	InitialHealth    string                       `json:"initial_health"`
	FinalHealth      string                       `json:"final_health"`
	FinalInstallMode string                       `json:"final_install_mode"`
	FinalInstallRoot string                       `json:"final_install_root"`
	Verification     *verifier.VerificationResult `json:"verification_result,omitempty"`
	Warnings         []string                     `json:"warnings"`
	Errors           []string                     `json:"errors"`
	ReportDir        string                       `json:"report_dir"`
	ExitCode         int                          `json:"exit_code"`
}

const (
	ExitInstallSuccess            = app.ExitSuccess
	ExitInstallWarnings           = app.ExitWarnings
	ExitInstallAdminRequired      = app.ExitAdminRequired
	ExitInstallInvalidInvite      = app.ExitInvalidInput
	ExitInstallInvalidInstaller   = app.ExitInvalidInput
	ExitInstallMSIFailed          = app.ExitInstallFailed
	ExitInstallVerificationFailed = app.ExitVerificationFailed
	ExitInstallRebootRequired     = app.ExitRebootRequired
	ExitInstallUnexpectedError    = app.ExitUnexpectedError
)
