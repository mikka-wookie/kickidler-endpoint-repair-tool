package preflight

import (
	"time"

	"kigrepair/internal/detector"
)

const (
	StatusReady             = "ready"
	StatusReadyWithWarnings = "ready_with_warnings"
	StatusNotReady          = "not_ready"
	StatusFailed            = "failed"

	CheckPass    = "pass"
	CheckWarning = "warning"
	CheckFail    = "fail"
	CheckSkipped = "skipped"
)

type Options struct {
	InstallerPath string
	HasInvite     bool
	OutputDir     string
	JSON          bool
	Quiet         bool
}

type Result struct {
	Command         string           `json:"command"`
	Status          string           `json:"status"`
	ReadyForRepair  bool             `json:"ready_for_repair"`
	ExitCode        int              `json:"exit_code"`
	Checks          []CheckResult    `json:"checks"`
	Installer       InstallerCheck   `json:"installer"`
	Environment     EnvironmentCheck `json:"environment"`
	DetectionHealth string           `json:"detection_health,omitempty"`
	InstallMode     string           `json:"install_mode,omitempty"`
	InstallRoot     string           `json:"install_root,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
	Errors          []string         `json:"errors,omitempty"`
	ReportDir       string           `json:"report_dir"`
	CreatedAt       string           `json:"created_at"`

	FailedRequiredChecks []string             `json:"failed_required_checks,omitempty"`
	Classification       ClassificationResult `json:"classification,omitempty"`
	Recommendation       RecommendationResult `json:"recommendation,omitempty"`
}

type CheckResult struct {
	Code        string `json:"code"`
	Status      string `json:"status"`
	Required    bool   `json:"required"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
	Action      string `json:"action,omitempty"`
}

type InstallerCheck struct {
	Required      bool   `json:"required"`
	Provided      bool   `json:"provided"`
	Discovered    bool   `json:"discovered"`
	Path          string `json:"path,omitempty"`
	Exists        bool   `json:"exists"`
	Readable      bool   `json:"readable"`
	SupportedName bool   `json:"supported_name"`
	SHA256        string `json:"sha256,omitempty"`
	Error         string `json:"error,omitempty"`
}

type EnvironmentCheck struct {
	IsAdmin          bool   `json:"is_admin"`
	OS               string `json:"os,omitempty"`
	Architecture     string `json:"architecture,omitempty"`
	ExePath          string `json:"exe_path,omitempty"`
	WorkingDir       string `json:"working_dir,omitempty"`
	PowerShellFound  bool   `json:"powershell_found"`
	MsiexecFound     bool   `json:"msiexec_found"`
	DefenderReadable bool   `json:"defender_readable"`
}

type ClassificationResult struct {
	Status       string                `json:"status"`
	PrimaryIssue *ClassificationIssue  `json:"primary_issue,omitempty"`
	Issues       []ClassificationIssue `json:"issues,omitempty"`
	Health       string                `json:"health,omitempty"`
}

type ClassificationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

type RecommendationResult struct {
	Status        string                 `json:"status"`
	PrimaryAction *RecommendationAction  `json:"primary_action,omitempty"`
	Actions       []RecommendationAction `json:"actions,omitempty"`
}

type RecommendationAction struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Command string `json:"command,omitempty"`
}

type Workflow struct {
	InstallerPath string
	HasInvite     bool

	Deps Dependencies
}

type Dependencies struct {
	Now              func() time.Time
	IsAdmin          func() bool
	OS               func() string
	Architecture     func() string
	ExePath          func() (string, error)
	WorkingDir       func() (string, error)
	LookPath         func(string) (string, error)
	ResolveInstaller func(string) InstallerResolution
	Detect           func() (detector.DetectionReport, error)
	ReadFile         func(string) error
	HashFile         func(string) (string, error)
	WriteProbe       func(string) error
}

type InstallerResolution struct {
	Path       string
	Provided   bool
	Discovered bool
	Error      string
}
