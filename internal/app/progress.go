package app

import "time"

type ProgressEvent struct {
	RunID           string          `json:"run_id,omitempty"`
	Workflow        string          `json:"workflow,omitempty"`
	OperationID     string          `json:"operation_id,omitempty"`
	Stage           string          `json:"stage"`
	Status          OperationStatus `json:"status,omitempty"`
	Message         string          `json:"message,omitempty"`
	Percent         int             `json:"percent,omitempty"`
	StartedAt       time.Time       `json:"started_at,omitempty"`
	FinishedAt      time.Time       `json:"finished_at,omitempty"`
	DurationMS      int64           `json:"duration_ms,omitempty"`
	ResultFile      string          `json:"result_file,omitempty"`
	FailureCategory string          `json:"failure_category,omitempty"`
}

type ProgressSink interface {
	OnProgress(event ProgressEvent)
}

type NoopProgressSink struct{}

func (NoopProgressSink) OnProgress(ProgressEvent) {}

type MemoryProgressSink struct {
	Events []ProgressEvent
}

func (s *MemoryProgressSink) OnProgress(event ProgressEvent) {
	if s == nil {
		return
	}
	s.Events = append(s.Events, event)
}
