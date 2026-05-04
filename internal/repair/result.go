package repair

import (
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/installer"
	"kigrepair/internal/verifier"
)

const (
	ExitSuccess              = app.ExitSuccess
	ExitWarnings             = app.ExitWarnings
	ExitAdminRequired        = app.ExitAdminRequired
	ExitConfirmationRequired = app.ExitConfirmationRequired
	ExitInvalidInput         = app.ExitInvalidInput
	ExitCleanupFailed        = app.ExitCleanupFailed
	ExitInstallFailed        = app.ExitInstallFailed
	ExitVerificationFailed   = app.ExitVerificationFailed
	ExitDefenderFailed       = app.ExitDefenderFailed
	ExitRebootRequired       = app.ExitRebootRequired
	ExitUnexpectedError      = app.ExitUnexpectedError
)

type RepairResult struct {
	StartedAt           time.Time                      `json:"started_at"`
	FinishedAt          time.Time                      `json:"finished_at"`
	Mode                string                         `json:"mode"`
	Force               bool                           `json:"force"`
	InviteProvided      bool                           `json:"invite_provided"`
	InstallerPath       string                         `json:"installer_path"`
	InstallerResolution *installer.InstallerResolution `json:"installer_resolution,omitempty"`
	InitialHealth       string                         `json:"initial_health"`
	InitialInstallMode  string                         `json:"initial_install_mode"`
	InitialInstallRoot  string                         `json:"initial_install_root"`
	CleanupNeeded       bool                           `json:"cleanup_needed"`
	CleanupExecuted     bool                           `json:"cleanup_executed"`
	InstallExecuted     bool                           `json:"install_executed"`
	DefenderExecuted    bool                           `json:"defender_executed"`
	FinalHealth         string                         `json:"final_health"`
	FinalInstallMode    string                         `json:"final_install_mode"`
	FinalInstallRoot    string                         `json:"final_install_root"`
	Verification        *verifier.VerificationResult   `json:"verification_result,omitempty"`
	RebootRequired      bool                           `json:"reboot_required"`
	Warnings            []string                       `json:"warnings"`
	Errors              []string                       `json:"errors"`
	ExitCode            int                            `json:"exit_code"`
	ReportDir           string                         `json:"report_dir"`
}
