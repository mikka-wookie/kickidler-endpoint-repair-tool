package wizard

import "kigrepair/internal/app"

const (
	StatusCompleted             = "completed"
	StatusCompletedWithWarnings = "completed_with_warnings"
	StatusRepairCompleted       = "repair_completed"
	StatusRepairFailed          = "repair_failed"
	StatusCancelled             = "cancelled"
	StatusNotReady              = "not_ready"
	StatusFailed                = "failed"
)

type Options struct {
	InstallerPath  string `json:"installer_path,omitempty"`
	HasInvite      bool   `json:"has_invite"`
	InviteValue    string `json:"-"`
	OutputDir      string `json:"output_dir,omitempty"`
	JSON           bool   `json:"json"`
	Quiet          bool   `json:"quiet"`
	NonInteractive bool   `json:"non_interactive"`
	CollectBundle  bool   `json:"collect_bundle"`
	AllowRepair    bool   `json:"allow_repair"`
	VerifyOnly     bool   `json:"verify_only"`
	Yes            bool   `json:"yes"`
}

type Result struct {
	Command            string       `json:"command"`
	Status             string       `json:"status"`
	ExitCode           int          `json:"exit_code"`
	ReportDir          string       `json:"report_dir"`
	Steps              []StepResult `json:"steps"`
	DetectionHealth    string       `json:"detection_health,omitempty"`
	InstallMode        string       `json:"install_mode,omitempty"`
	InstallRoot        string       `json:"install_root,omitempty"`
	PrimaryService     string       `json:"primary_service,omitempty"`
	ClassificationCode string       `json:"classification_code,omitempty"`
	RecommendationCode string       `json:"recommendation_code,omitempty"`
	PreflightStatus    string       `json:"preflight_status,omitempty"`
	RepairPlanStatus   string       `json:"repair_plan_status,omitempty"`
	RepairExecuted     bool         `json:"repair_executed"`
	VerifyStatus       string       `json:"verify_status,omitempty"`
	BundlePath         string       `json:"bundle_path,omitempty"`
	Warnings           []string     `json:"warnings,omitempty"`
	Errors             []string     `json:"errors,omitempty"`
}

type StepResult struct {
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary,omitempty"`
	Artifacts []string `json:"artifacts,omitempty"`
}

func ExitCode(status string) int {
	switch status {
	case StatusCompleted, StatusRepairCompleted:
		return app.ExitSuccess
	case StatusCompletedWithWarnings, StatusCancelled:
		return app.ExitWarnings
	case StatusNotReady, StatusRepairFailed:
		return app.ExitVerificationFailed
	default:
		return app.ExitUnexpectedError
	}
}
