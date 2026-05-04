package diagnostics

import "time"

type CollectReportResult struct {
	StartedAt   time.Time         `json:"started_at"`
	FinishedAt  time.Time         `json:"finished_at"`
	Mode        string            `json:"mode"`
	ReportDir   string            `json:"report_dir"`
	BundlePath  string            `json:"bundle_path,omitempty"`
	Collectors  []CollectorResult `json:"collectors"`
	Warnings    []string          `json:"warnings"`
	Errors      []string          `json:"errors"`
	Health      string            `json:"health,omitempty"`
	InstallMode string            `json:"install_mode,omitempty"`
	InstallRoot string            `json:"install_root,omitempty"`
	ExitCode    int               `json:"exit_code"`
}

type CollectorResult struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	OutputFiles []string `json:"output_files"`
	Message     string   `json:"message,omitempty"`
	Error       string   `json:"error,omitempty"`
}
