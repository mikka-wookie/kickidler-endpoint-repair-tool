package app

import (
	"time"

	"kigrepair/internal/config"
)

type Logger interface {
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Close() error
}

type Reporter interface {
	WriteJSON(name string, v any) error
	WriteText(name string, content string) error
	WriteOperations(results []OperationResult) error
	Archive() (string, error)
}

type AppContext struct {
	Mode           RunMode
	OutputDir      string
	ReportRoot     string
	StartedAt      time.Time
	Quiet          bool
	NonInteractive bool
	Force          bool
	JSONOutput     bool
	Config         config.Config
	ConfigMeta     config.Metadata
	Logger         Logger
	Reporter       Reporter
	Results        []OperationResult
	JSONValue      any
	ExitCode       int
}

func NewContext() *AppContext {
	startedAt := time.Now()
	return &AppContext{
		Mode:       RunModeCLI,
		ReportRoot: config.DefaultReportRoot,
		Config:     config.DefaultConfig(),
		ConfigMeta: config.EffectiveConfig{Config: config.DefaultConfig()}.Metadata(),
		StartedAt:  startedAt,
		Results:    make([]OperationResult, 0),
	}
}

func (c *AppContext) AddResult(result OperationResult) {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = result.Timestamp
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = result.Timestamp
	}
	if result.DurationMS == 0 && !result.StartedAt.IsZero() && !result.FinishedAt.IsZero() {
		result.DurationMS = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
	}
	c.Results = append(c.Results, result)
}
