package installer

import "time"

type InstallResult struct {
	StartedAt        time.Time `json:"started_at"`
	FinishedAt       time.Time `json:"finished_at"`
	InstallerPath    string    `json:"installer_path"`
	InviteProvided   bool      `json:"invite_provided"`
	MSI              MSIResult `json:"msi"`
	InitialHealth    string    `json:"initial_health"`
	FinalHealth      string    `json:"final_health"`
	FinalInstallMode string    `json:"final_install_mode"`
	FinalInstallRoot string    `json:"final_install_root"`
	Warnings         []string  `json:"warnings"`
	Errors           []string  `json:"errors"`
	ReportDir        string    `json:"report_dir"`
	ExitCode         int       `json:"exit_code"`
}

const (
	ExitInstallSuccess            = 0
	ExitInstallWarnings           = 1
	ExitInstallAdminRequired      = 2
	ExitInstallInvalidInvite      = 3
	ExitInstallInvalidInstaller   = 4
	ExitInstallMSIFailed          = 5
	ExitInstallVerificationFailed = 6
	ExitInstallRebootRequired     = 9
	ExitInstallUnexpectedError    = 10
)
