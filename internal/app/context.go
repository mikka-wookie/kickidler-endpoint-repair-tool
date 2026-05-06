package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/config"
	"kigrepair/internal/version"
)

type Logger interface {
	Debug(format string, args ...any)
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
	Run            RunMetadata
	Progress       ProgressSink
	operationSeq   int
}

func NewContext() *AppContext {
	startedAt := time.Now()
	runID := NewRunID(startedAt)
	info := version.Get()
	return &AppContext{
		Mode:       RunModeCLI,
		ReportRoot: config.DefaultReportRoot,
		Config:     config.DefaultConfig(),
		ConfigMeta: config.EffectiveConfig{Config: config.DefaultConfig()}.Metadata(),
		StartedAt:  startedAt,
		Results:    make([]OperationResult, 0),
		Run: RunMetadata{
			RunID:           runID,
			CorrelationID:   runID,
			StartedAt:       startedAt,
			Version:         info.Version,
			Commit:          info.Commit,
			BuildDate:       info.BuildDate,
			UserInteractive: true,
		},
		Progress: NoopProgressSink{},
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
	if result.ID == "" {
		result.ID = c.nextOperationID(result.Step)
	}
	if result.RunID == "" {
		result.RunID = c.Run.RunID
	}
	if result.Workflow == "" {
		result.Workflow = c.Run.WorkflowName
	}
	if result.Name == "" {
		result.Name = result.Step
	}
	if result.Category == "" {
		result.Category = inferOperationCategory(result.Step)
	}
	if result.FailureCategory == "" && result.Status == OperationStatusFailed {
		result.FailureCategory = result.Category
	}
	result.ReadOnly = result.ReadOnly || c.Run.ReadOnly
	result.DryRun = result.DryRun || c.Run.DryRun
	c.Results = append(c.Results, result)
	c.emitProgress(result, "operation_started")
	c.emitProgress(result, "operation_finished")
}

func (c *AppContext) FinishRun() {
	c.Run.FinishedAt = time.Now()
	c.Run.DurationMS = c.Run.FinishedAt.Sub(c.Run.StartedAt).Milliseconds()
}

func (c *AppContext) nextOperationID(step string) string {
	c.operationSeq++
	return fmt.Sprintf("op-%03d-%s", c.operationSeq, Slug(shortStep(step)))
}

func (c *AppContext) emitProgress(result OperationResult, stage string) {
	if c.Progress == nil {
		return
	}
	event := ProgressEvent{
		RunID:           result.RunID,
		Workflow:        result.Workflow,
		OperationID:     result.ID,
		Stage:           stage,
		Status:          result.Status,
		Message:         result.Message,
		StartedAt:       result.StartedAt,
		FinishedAt:      result.FinishedAt,
		DurationMS:      result.DurationMS,
		ResultFile:      result.ResultFile,
		FailureCategory: result.FailureCategory,
	}
	c.Progress.OnProgress(event)
}

func shortStep(step string) string {
	step = strings.TrimSpace(step)
	if step == "" {
		return "operation"
	}
	step = filepath.Base(strings.ReplaceAll(step, ".", string(filepath.Separator)))
	if step == "" || step == "." {
		return "operation"
	}
	return step
}

func inferOperationCategory(step string) string {
	lower := strings.ToLower(step)
	switch {
	case strings.Contains(lower, "detect"):
		return OperationCategoryDetection
	case strings.Contains(lower, "classif"):
		return OperationCategoryClassification
	case strings.Contains(lower, "recommend"):
		return OperationCategoryRecommendation
	case strings.Contains(lower, "preflight"):
		return OperationCategoryPreflight
	case strings.Contains(lower, "cleanup"):
		return OperationCategoryCleanup
	case strings.Contains(lower, "install") || strings.Contains(lower, "msi") || strings.Contains(lower, "validation"):
		return OperationCategoryInstaller
	case strings.Contains(lower, "defender"):
		return OperationCategoryDefender
	case strings.Contains(lower, "verify") || strings.Contains(lower, "verification"):
		return OperationCategoryVerification
	case strings.Contains(lower, "rollback"):
		return OperationCategoryRollback
	case strings.Contains(lower, "collect") || strings.Contains(lower, "diagnostic") || strings.Contains(lower, "bundle"):
		return OperationCategoryDiagnostics
	case strings.Contains(lower, "report"):
		return OperationCategoryReports
	case strings.Contains(lower, "config"):
		return OperationCategoryConfig
	case strings.Contains(lower, "command") || strings.Contains(lower, "powershell") || strings.Contains(lower, "external"):
		return OperationCategoryExternal
	case strings.Contains(lower, "workflow"):
		return OperationCategoryWorkflow
	default:
		return OperationCategoryUnknown
	}
}
