package diagnostics

import (
	"time"

	"kigrepair/internal/recommendations"
)

type CollectReportResult struct {
	StartedAt        time.Time                             `json:"started_at"`
	FinishedAt       time.Time                             `json:"finished_at"`
	Mode             string                                `json:"mode"`
	Status           string                                `json:"status"`
	ReportDir        string                                `json:"report_dir"`
	BundlePath       string                                `json:"bundle_path,omitempty"`
	Collectors       []CollectorResult                     `json:"collectors"`
	FilesIncluded    []CollectedFile                       `json:"files_included"`
	FilesSkipped     []string                              `json:"files_skipped"`
	Warnings         []string                              `json:"warnings"`
	Errors           []string                              `json:"errors"`
	RedactionEnabled bool                                  `json:"redaction_enabled"`
	Health           string                                `json:"health,omitempty"`
	InstallMode      string                                `json:"install_mode,omitempty"`
	InstallRoot      string                                `json:"install_root,omitempty"`
	Recommendation   *recommendations.RecommendationResult `json:"recommendation,omitempty"`
	ExitCode         int                                   `json:"exit_code"`
}

type CollectorResult struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	OutputFiles []string `json:"output_files"`
	Message     string   `json:"message,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type CollectedFile struct {
	Path      string `json:"path"`
	Source    string `json:"source"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}
