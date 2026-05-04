package repair

import "time"

const (
	ExitSuccess              = 0
	ExitWarnings             = 1
	ExitAdminRequired        = 2
	ExitConfirmationRequired = 3
	ExitInvalidInput         = 4
	ExitCleanupFailed        = 5
	ExitInstallFailed        = 6
	ExitVerificationFailed   = 7
	ExitDefenderFailed       = 8
	ExitRebootRequired       = 9
	ExitUnexpectedError      = 10
)

type RepairResult struct {
	StartedAt          time.Time `json:"started_at"`
	FinishedAt         time.Time `json:"finished_at"`
	Mode               string    `json:"mode"`
	Force              bool      `json:"force"`
	InviteProvided     bool      `json:"invite_provided"`
	InstallerPath      string    `json:"installer_path"`
	InitialHealth      string    `json:"initial_health"`
	InitialInstallMode string    `json:"initial_install_mode"`
	InitialInstallRoot string    `json:"initial_install_root"`
	CleanupNeeded      bool      `json:"cleanup_needed"`
	CleanupExecuted    bool      `json:"cleanup_executed"`
	InstallExecuted    bool      `json:"install_executed"`
	DefenderExecuted   bool      `json:"defender_executed"`
	FinalHealth        string    `json:"final_health"`
	FinalInstallMode   string    `json:"final_install_mode"`
	FinalInstallRoot   string    `json:"final_install_root"`
	RebootRequired     bool      `json:"reboot_required"`
	Warnings           []string  `json:"warnings"`
	Errors             []string  `json:"errors"`
	ExitCode           int       `json:"exit_code"`
	ReportDir          string    `json:"report_dir"`
}
