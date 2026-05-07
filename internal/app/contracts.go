package app

import (
	"context"
	"time"

	"kigrepair/internal/config"
)

type WorkflowStatus string

const (
	WorkflowStatusIdle      WorkflowStatus = "idle"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusSuccess   WorkflowStatus = "success"
	WorkflowStatusWarning   WorkflowStatus = "warning"
	WorkflowStatusNotReady  WorkflowStatus = "not_ready"
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
	Outcome        *WorkflowOutcome     `json:"outcome,omitempty"`
	Result         any                  `json:"result,omitempty"`
	Classification any                  `json:"classification,omitempty"`
	Recommendation any                  `json:"recommendation,omitempty"`
	Warnings       []string             `json:"warnings,omitempty"`
	Errors         []string             `json:"errors,omitempty"`
	Timeline       []OperationResult    `json:"timeline,omitempty"`
	Policy         config.PolicySummary `json:"policy"`
}

type WorkflowOutcome struct {
	RunID             string               `json:"run_id"`
	Workflow          string               `json:"workflow"`
	Status            string               `json:"status"`
	ExitCode          int                  `json:"exit_code"`
	StartedAt         time.Time            `json:"started_at"`
	FinishedAt        time.Time            `json:"finished_at"`
	DurationMS        int64                `json:"duration_ms"`
	ReportDir         string               `json:"report_dir,omitempty"`
	SummaryFile       string               `json:"summary_file,omitempty"`
	OperationsFile    string               `json:"operations_file,omitempty"`
	PrimaryResultFile string               `json:"primary_result_file,omitempty"`
	Health            string               `json:"health,omitempty"`
	InstallMode       string               `json:"install_mode,omitempty"`
	RepairReadiness   string               `json:"repair_readiness,omitempty"`
	PrimaryIssue      *IssueSummary        `json:"primary_issue,omitempty"`
	Recommendation    *ActionSummary       `json:"recommendation,omitempty"`
	BlockingReasons   []UserMessage        `json:"blocking_reasons,omitempty"`
	Warnings          []UserMessage        `json:"warnings,omitempty"`
	Errors            []UserMessage        `json:"errors,omitempty"`
	Timeline          []TimelineItem       `json:"timeline,omitempty"`
	Files             ResultFileSummary    `json:"files,omitempty"`
	Policy            config.PolicySummary `json:"policy,omitempty"`
	Admin             AdminSummary         `json:"admin,omitempty"`
	Installer         InstallerSummary     `json:"installer,omitempty"`
	Defender          DefenderSummary      `json:"defender,omitempty"`
	Extra             map[string]string    `json:"extra,omitempty"`
}

type UserMessage struct {
	Code       string `json:"code,omitempty"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	Action     string `json:"action,omitempty"`
	DetailsRef string `json:"details_ref,omitempty"`
}

type TimelineItem struct {
	OperationID     string `json:"operation_id,omitempty"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	DurationMS      int64  `json:"duration_ms,omitempty"`
	FailureCategory string `json:"failure_category,omitempty"`
	DetailsRef      string `json:"details_ref,omitempty"`
}

type IssueSummary struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Action   string `json:"action,omitempty"`
}

type ActionSummary struct {
	Code          string `json:"code"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Command       string `json:"command,omitempty"`
	Destructive   bool   `json:"destructive"`
	RequiresAdmin bool   `json:"requires_admin"`
	RequiresYes   bool   `json:"requires_yes"`
}

type ResultFileSummary struct {
	ReportDir            string `json:"report_dir,omitempty"`
	Summary              string `json:"summary,omitempty"`
	Operations           string `json:"operations,omitempty"`
	InitialDetection     string `json:"initial_detection,omitempty"`
	FinalDetection       string `json:"final_detection,omitempty"`
	PreflightResult      string `json:"preflight_result,omitempty"`
	RepairPlan           string `json:"repair_plan,omitempty"`
	RepairResult         string `json:"repair_result,omitempty"`
	CleanupPlan          string `json:"cleanup_plan,omitempty"`
	CleanupResult        string `json:"cleanup_result,omitempty"`
	VerificationResult   string `json:"verification_result,omitempty"`
	ClassificationResult string `json:"classification_result,omitempty"`
	RecommendationResult string `json:"recommendation_result,omitempty"`
	InstallerValidation  string `json:"installer_validation,omitempty"`
	DefenderResult       string `json:"defender_result,omitempty"`
	SupportBundle        string `json:"support_bundle,omitempty"`
	ReportsList          string `json:"reports_list,omitempty"`
	ReportsCleanupPlan   string `json:"reports_cleanup_plan,omitempty"`
	ReportsCleanupResult string `json:"reports_cleanup_result,omitempty"`
	ConfigShow           string `json:"config_show,omitempty"`
	ConfigValidation     string `json:"config_validation,omitempty"`
	PrimaryResult        string `json:"primary_result,omitempty"`
}

type AdminSummary struct {
	IsAdmin     bool   `json:"is_admin"`
	LimitedMode bool   `json:"limited_mode"`
	Message     string `json:"message,omitempty"`
}

type InstallerSummary struct {
	Path     string `json:"path,omitempty"`
	Exists   bool   `json:"exists"`
	Readable bool   `json:"readable"`
	Valid    bool   `json:"valid"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
}

type DefenderSummary struct {
	Status   string `json:"status,omitempty"`
	Required bool   `json:"required"`
	Covered  bool   `json:"covered"`
	Message  string `json:"message,omitempty"`
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
