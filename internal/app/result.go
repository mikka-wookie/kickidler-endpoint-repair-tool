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
	Step       string          `json:"step"`
	Target     string          `json:"target"`
	Status     OperationStatus `json:"status"`
	Message    string          `json:"message"`
	Error      string          `json:"error,omitempty"`
	Category   string          `json:"category,omitempty"`
	StartedAt  time.Time       `json:"started_at,omitempty"`
	FinishedAt time.Time       `json:"finished_at,omitempty"`
	DurationMS int64           `json:"duration_ms,omitempty"`
	Artifact   string          `json:"artifact,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}
