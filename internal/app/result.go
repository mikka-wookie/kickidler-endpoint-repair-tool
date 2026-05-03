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
	Step      string          `json:"step"`
	Target    string          `json:"target"`
	Status    OperationStatus `json:"status"`
	Message   string          `json:"message"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}
