package services

import "time"

const (
	TrustTrusted           = "trusted"
	TrustSupportedNameOnly = "supported_name_only"
	TrustPathMismatch      = "path_mismatch"
	TrustUnknown           = "unknown"
	TrustQueryFailed       = "query_failed"
)

const (
	StateRunning         = "running"
	StateStopped         = "stopped"
	StateStartPending    = "start_pending"
	StateStopPending     = "stop_pending"
	StatePaused          = "paused"
	StatePausePending    = "pause_pending"
	StateContinuePending = "continue_pending"
	StateUnknown         = "unknown"
)

const (
	ModeUnknown   = "unknown"
	ModeStandard  = "standard"
	ModeHelper    = "helper"
	ModeHiddenWMI = "hidden_wmi"
)

type CommandResult struct {
	ExitCode int
	Output   string
	Err      error
}

type CommandRunner func(name string, args ...string) CommandResult

type ServiceInfo struct {
	Name                     string   `json:"name"`
	Exists                   bool     `json:"exists"`
	State                    string   `json:"state,omitempty"`
	StartType                string   `json:"start_type,omitempty"`
	RawImagePath             string   `json:"raw_image_path,omitempty"`
	ExecutablePath           string   `json:"executable_path,omitempty"`
	NormalizedExecutablePath string   `json:"normalized_executable_path,omitempty"`
	Arguments                []string `json:"arguments,omitempty"`
	InstallRoot              string   `json:"install_root,omitempty"`
	InstallMode              string   `json:"install_mode,omitempty"`
	GrabberRelated           bool     `json:"grabber_related"`
	TrustLevel               string   `json:"trust_level"`
	Warnings                 []string `json:"warnings,omitempty"`
	Errors                   []string `json:"errors,omitempty"`
}

type ParsedImagePath struct {
	Raw            string   `json:"raw"`
	ExecutablePath string   `json:"executable_path"`
	NormalizedPath string   `json:"normalized_path"`
	Arguments      []string `json:"arguments,omitempty"`
	Valid          bool     `json:"valid"`
	Warnings       []string `json:"warnings,omitempty"`
	Error          string   `json:"error,omitempty"`
}

type ServiceActionResult struct {
	Action      string   `json:"action"`
	ServiceName string   `json:"service_name"`
	Status      string   `json:"status"`
	BeforeState string   `json:"before_state,omitempty"`
	AfterState  string   `json:"after_state,omitempty"`
	Trusted     bool     `json:"trusted"`
	Message     string   `json:"message,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	Errors      []string `json:"errors,omitempty"`
}

type StopOptions struct {
	Runner        CommandRunner
	Timeout       time.Duration
	PollInterval  time.Duration
	AllowNameOnly bool
}

type DeleteOptions struct {
	Runner        CommandRunner
	Timeout       time.Duration
	PollInterval  time.Duration
	AllowStop     bool
	AllowNameOnly bool
}

const (
	DefaultStopTimeout         = 30 * time.Second
	DefaultDeleteVerifyTimeout = 10 * time.Second
	DefaultQueryRetryInterval  = 500 * time.Millisecond
)
