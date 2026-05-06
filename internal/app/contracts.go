package app

import (
	"context"
	"time"

	"kigrepair/internal/config"
)

type WorkflowStatus string

const (
	WorkflowStatusSuccess   WorkflowStatus = "success"
	WorkflowStatusWarning   WorkflowStatus = "warning"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

type CommonRequest struct {
	ConfigPath     string `json:"config_path,omitempty"`
	Profile        string `json:"profile,omitempty"`
	OutputDir      string `json:"output_dir,omitempty"`
	JSONOutput     bool   `json:"json_output,omitempty"`
	Quiet          bool   `json:"quiet,omitempty"`
	NonInteractive bool   `json:"non_interactive,omitempty"`
	Force          bool   `json:"force,omitempty"`
	Yes            bool   `json:"yes,omitempty"`
	DryRun         bool   `json:"dry_run,omitempty"`
	LogLevel       string `json:"log_level,omitempty"`
}

type InstallerRequestFields struct {
	InstallerPath string `json:"installer_path,omitempty"`
	InviteValue   string `json:"-"`
}

type CheckRequest struct {
	CommonRequest
}

type VerifyRequest struct {
	CommonRequest
}

type PreflightRequest struct {
	CommonRequest
	InstallerRequestFields
}

type RepairPlanRequest struct {
	CommonRequest
	InstallerRequestFields
}

type RepairRequest struct {
	CommonRequest
	InstallerRequestFields
}

type CleanupPlanRequest struct {
	CommonRequest
}

type CleanupRequest struct {
	CommonRequest
}

type CollectReportRequest struct {
	CommonRequest
	IncludeEventLogs bool `json:"include_eventlogs,omitempty"`
	IncludeHistory   bool `json:"include_history,omitempty"`
	HistoryLimit     int  `json:"history_limit,omitempty"`
	NoZip            bool `json:"no_zip,omitempty"`
}

type ReportsListRequest struct {
	CommonRequest
	Limit int  `json:"limit,omitempty"`
	All   bool `json:"all,omitempty"`
}

type ReportsCleanupPlanRequest struct {
	CommonRequest
	OlderThan string `json:"older_than,omitempty"`
	KeepLast  int    `json:"keep_last,omitempty"`
}

type ReportsCleanupRequest struct {
	CommonRequest
	OlderThan string `json:"older_than,omitempty"`
	KeepLast  int    `json:"keep_last,omitempty"`
}

type ConfigShowRequest struct {
	CommonRequest
}

type ConfigValidateRequest struct {
	CommonRequest
}

type WorkflowResponseMeta struct {
	RunID             string    `json:"run_id"`
	Workflow          string    `json:"workflow"`
	Command           string    `json:"command"`
	Status            string    `json:"status"`
	ExitCode          int       `json:"exit_code"`
	ReportDir         string    `json:"report_dir"`
	SummaryFile       string    `json:"summary_file,omitempty"`
	OperationsFile    string    `json:"operations_file,omitempty"`
	PrimaryResultFile string    `json:"primary_result_file,omitempty"`
	StartedAt         time.Time `json:"started_at"`
	FinishedAt        time.Time `json:"finished_at"`
	DurationMS        int64     `json:"duration_ms"`
}

type WorkflowResponse struct {
	Meta           WorkflowResponseMeta `json:"meta"`
	Result         any                  `json:"result,omitempty"`
	Classification any                  `json:"classification,omitempty"`
	Recommendation any                  `json:"recommendation,omitempty"`
	Warnings       []string             `json:"warnings,omitempty"`
	Errors         []string             `json:"errors,omitempty"`
	Timeline       []OperationResult    `json:"timeline,omitempty"`
	Policy         config.PolicySummary `json:"policy"`
}

type CheckResponse = WorkflowResponse
type VerifyResponse = WorkflowResponse
type PreflightResponse = WorkflowResponse
type RepairPlanResponse = WorkflowResponse
type RepairResponse = WorkflowResponse
type CleanupPlanResponse = WorkflowResponse
type CleanupResponse = WorkflowResponse
type CollectReportResponse = WorkflowResponse
type ReportsListResponse = WorkflowResponse
type ReportsCleanupPlanResponse = WorkflowResponse
type ReportsCleanupResponse = WorkflowResponse
type ConfigShowResponse = WorkflowResponse
type ConfigValidateResponse = WorkflowResponse

type WorkflowOptions struct {
	ProgressSink ProgressSink
	ConfirmSink  ConfirmSink
}

type ConfirmationPrompt struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Message         string `json:"message"`
	RequiredText    string `json:"required_text,omitempty"`
	Destructive     bool   `json:"destructive"`
	DefaultDecision string `json:"default_decision,omitempty"`
}

type ConfirmationResponse struct {
	Accepted bool   `json:"accepted"`
	Text     string `json:"-"`
}

type ConfirmSink interface {
	Confirm(ctx context.Context, prompt ConfirmationPrompt) (ConfirmationResponse, error)
}

type WorkflowService interface {
	Check(ctx context.Context, req CheckRequest) (*CheckResponse, error)
	Verify(ctx context.Context, req VerifyRequest) (*VerifyResponse, error)
	Preflight(ctx context.Context, req PreflightRequest) (*PreflightResponse, error)
	RepairPlan(ctx context.Context, req RepairPlanRequest) (*RepairPlanResponse, error)
	Repair(ctx context.Context, req RepairRequest) (*RepairResponse, error)
	CleanupPlan(ctx context.Context, req CleanupPlanRequest) (*CleanupPlanResponse, error)
	Cleanup(ctx context.Context, req CleanupRequest) (*CleanupResponse, error)
	CollectReport(ctx context.Context, req CollectReportRequest) (*CollectReportResponse, error)
	ReportsList(ctx context.Context, req ReportsListRequest) (*ReportsListResponse, error)
	ReportsCleanupPlan(ctx context.Context, req ReportsCleanupPlanRequest) (*ReportsCleanupPlanResponse, error)
	ReportsCleanup(ctx context.Context, req ReportsCleanupRequest) (*ReportsCleanupResponse, error)
	ConfigShow(ctx context.Context, req ConfigShowRequest) (*ConfigShowResponse, error)
	ConfigValidate(ctx context.Context, req ConfigValidateRequest) (*ConfigValidateResponse, error)
	GetWorkflowCatalog(ctx context.Context) WorkflowCatalog
}
