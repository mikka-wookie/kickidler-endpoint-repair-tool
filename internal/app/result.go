package app

import "time"

type OperationStatus string

const (
	OperationStatusSuccess OperationStatus = "success"
	OperationStatusFailed  OperationStatus = "failed"
	OperationStatusSkipped OperationStatus = "skipped"
	OperationStatusWarning OperationStatus = "warning"
)

type OperationResult struct {
	ID              string          `json:"id,omitempty"`
	RunID           string          `json:"run_id,omitempty"`
	Workflow        string          `json:"workflow,omitempty"`
	Name            string          `json:"name,omitempty"`
	Step            string          `json:"step"`
	Target          string          `json:"target"`
	Status          OperationStatus `json:"status"`
	Message         string          `json:"message"`
	Details         string          `json:"details,omitempty"`
	FailureCategory string          `json:"failure_category,omitempty"`
	Error           string          `json:"error,omitempty"`
	Category        string          `json:"category,omitempty"`
	StartedAt       time.Time       `json:"started_at,omitempty"`
	FinishedAt      time.Time       `json:"finished_at,omitempty"`
	DurationMS      int64           `json:"duration_ms,omitempty"`
	Artifact        string          `json:"artifact,omitempty"`
	RedactedCommand string          `json:"redacted_command,omitempty"`
	ResultFile      string          `json:"result_file,omitempty"`
	RelatedFiles    []string        `json:"related_files,omitempty"`
	ReadOnly        bool            `json:"read_only,omitempty"`
	DryRun          bool            `json:"dry_run,omitempty"`
	Timestamp       time.Time       `json:"timestamp"`
}

const (
	OperationCategoryDetection      = "detection"
	OperationCategoryClassification = "classification"
	OperationCategoryRecommendation = "recommendation"
	OperationCategoryPreflight      = "preflight"
	OperationCategoryCleanup        = "cleanup"
	OperationCategoryInstaller      = "installer"
	OperationCategoryDefender       = "defender"
	OperationCategoryVerification   = "verification"
	OperationCategoryRollback       = "rollback"
	OperationCategoryDiagnostics    = "diagnostics"
	OperationCategoryReports        = "reports"
	OperationCategoryConfig         = "config"
	OperationCategoryExternal       = "external_command"
	OperationCategoryWorkflow       = "workflow"
	OperationCategoryUnknown        = "unknown"
)
