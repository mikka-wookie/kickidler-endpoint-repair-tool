package workflowservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/config"
	"kigrepair/internal/diagnostics"
	"kigrepair/internal/logging"
	"kigrepair/internal/preflight"
	"kigrepair/internal/repair"
	"kigrepair/internal/reports"
	"kigrepair/internal/verifier"
)

type Service struct {
	Options app.WorkflowOptions
}

func New(options app.WorkflowOptions) *Service {
	return &Service{Options: options}
}

func NewDefault() *Service {
	return New(app.WorkflowOptions{})
}

func (s *Service) Check(ctx context.Context, req app.CheckRequest) (*app.CheckResponse, error) {
	return s.execute(ctx, req.CommonRequest, checks.CheckWorkflow{}, "initial-detection.json", nil)
}

func (s *Service) Verify(ctx context.Context, req app.VerifyRequest) (*app.VerifyResponse, error) {
	return s.execute(ctx, req.CommonRequest, verifier.VerifyWorkflow{}, "verification-result.json", nil)
}

func (s *Service) Preflight(ctx context.Context, req app.PreflightRequest) (*app.PreflightResponse, error) {
	return s.execute(ctx, req.CommonRequest, preflight.Workflow{
		InstallerPath: req.InstallerPath,
		HasInvite:     strings.TrimSpace(req.InviteValue) != "",
	}, "preflight-result.json", []string{req.InviteValue})
}

func (s *Service) RepairPlan(ctx context.Context, req app.RepairPlanRequest) (*app.RepairPlanResponse, error) {
	common := req.CommonRequest
	common.DryRun = true
	return s.execute(ctx, common, repair.DryRunWorkflow{
		Invite:    req.InviteValue,
		Installer: req.InstallerPath,
		Yes:       req.Yes,
	}, "repair-plan.json", []string{req.InviteValue})
}

func (s *Service) Repair(ctx context.Context, req app.RepairRequest) (*app.RepairResponse, error) {
	return s.execute(ctx, req.CommonRequest, repair.RepairWorkflow{
		Invite:    req.InviteValue,
		Installer: req.InstallerPath,
		Yes:       req.Yes,
	}, "repair-result.json", []string{req.InviteValue})
}

func (s *Service) CleanupPlan(ctx context.Context, req app.CleanupPlanRequest) (*app.CleanupPlanResponse, error) {
	common := req.CommonRequest
	common.DryRun = true
	return s.execute(ctx, common, cleaner.CleanupWorkflow{DryRun: true, Yes: req.Yes}, "cleanup-plan.json", nil)
}

func (s *Service) Cleanup(ctx context.Context, req app.CleanupRequest) (*app.CleanupResponse, error) {
	return s.execute(ctx, req.CommonRequest, cleaner.CleanupWorkflow{DryRun: req.DryRun, Yes: req.Yes}, "cleanup-result.json", nil)
}

func (s *Service) CollectReport(ctx context.Context, req app.CollectReportRequest) (*app.CollectReportResponse, error) {
	return s.execute(ctx, req.CommonRequest, diagnostics.CollectReportWorkflow{
		IncludeEventLogs: req.IncludeEventLogs,
		IncludeHistory:   req.IncludeHistory,
		HistoryLimit:     req.HistoryLimit,
		NoZip:            req.NoZip,
	}, "collect-result.json", nil)
}

func (s *Service) ReportsList(ctx context.Context, req app.ReportsListRequest) (*app.ReportsListResponse, error) {
	resp, runCtx, err := s.prepare(ctx, req.CommonRequest, "reports list", true)
	if err != nil {
		return resp, err
	}
	if cancelled := s.cancelled(ctx, runCtx, resp); cancelled {
		return resp, nil
	}
	result, listErr := reports.ListReports(runCtx.ReportRoot, reports.ReportListOptions{Limit: req.Limit, All: req.All})
	if listErr != nil {
		resp.Errors = append(resp.Errors, listErr.Error())
		runCtx.ExitCode = app.ExitUnexpectedError
	}
	if len(result.Warnings) > 0 {
		resp.Warnings = append(resp.Warnings, result.Warnings...)
		if runCtx.ExitCode == app.ExitSuccess {
			runCtx.ExitCode = app.ExitWarnings
		}
	}
	runCtx.JSONValue = result
	_ = runCtx.Reporter.WriteJSON("reports-list", result)
	_ = runCtx.Reporter.WriteText("summary", reports.FormatReportList(result))
	return s.finish(runCtx, resp, "reports-list.json", listErr)
}

func (s *Service) ReportsCleanupPlan(ctx context.Context, req app.ReportsCleanupPlanRequest) (*app.ReportsCleanupPlanResponse, error) {
	common := req.CommonRequest
	common.DryRun = true
	return s.execute(ctx, common, reports.CleanupReportsWorkflow{
		OlderThan: req.OlderThan,
		KeepLast:  req.KeepLast,
		DryRun:    true,
		Yes:       req.Yes,
	}, "report-cleanup-plan.json", nil)
}

func (s *Service) ReportsCleanup(ctx context.Context, req app.ReportsCleanupRequest) (*app.ReportsCleanupResponse, error) {
	return s.execute(ctx, req.CommonRequest, reports.CleanupReportsWorkflow{
		OlderThan: req.OlderThan,
		KeepLast:  req.KeepLast,
		DryRun:    req.DryRun,
		Yes:       req.Yes,
	}, "report-cleanup-result.json", nil)
}

func (s *Service) ConfigShow(ctx context.Context, req app.ConfigShowRequest) (*app.ConfigShowResponse, error) {
	resp, runCtx, err := s.prepare(ctx, req.CommonRequest, "config show", true)
	if err != nil {
		return resp, err
	}
	if cancelled := s.cancelled(ctx, runCtx, resp); cancelled {
		return resp, nil
	}
	effective := config.EffectiveConfig{Config: runCtx.Config, Path: runCtx.ConfigMeta.Path, Warnings: runCtx.ConfigMeta.Warnings}
	runCtx.JSONValue = effective
	_ = runCtx.Reporter.WriteJSON("config-show", effective)
	_ = runCtx.Reporter.WriteText("summary", config.FormatShow(effective))
	return s.finish(runCtx, resp, "config-show.json", nil)
}

func (s *Service) ConfigValidate(ctx context.Context, req app.ConfigValidateRequest) (*app.ConfigValidateResponse, error) {
	resp, runCtx, err := s.prepare(ctx, req.CommonRequest, "config validate", true)
	if err != nil {
		return resp, err
	}
	if cancelled := s.cancelled(ctx, runCtx, resp); cancelled {
		return resp, nil
	}
	validation := config.ValidateConfig(runCtx.Config)
	validation.Warnings = append(runCtx.ConfigMeta.Warnings, validation.Warnings...)
	if !validation.OK() {
		runCtx.ExitCode = app.ExitInvalidInput
	}
	runCtx.JSONValue = validation
	_ = runCtx.Reporter.WriteJSON("config-validation", validation)
	_ = runCtx.Reporter.WriteText("summary", config.FormatValidation(validation))
	return s.finish(runCtx, resp, "config-validation.json", nil)
}

func (s *Service) GetWorkflowCatalog(context.Context) app.WorkflowCatalog {
	return app.DefaultWorkflowCatalog()
}

func (s *Service) execute(ctx context.Context, req app.CommonRequest, workflow app.Workflow, primary string, sensitive []string) (*app.WorkflowResponse, error) {
	return s.ExecuteWorkflowWithSensitive(ctx, req, workflow, primary, sensitive)
}

func (s *Service) ExecuteWorkflow(ctx context.Context, req app.CommonRequest, workflow app.Workflow, primary string) (*app.WorkflowResponse, error) {
	return s.ExecuteWorkflowWithSensitive(ctx, req, workflow, primary, nil)
}

func (s *Service) ExecuteWorkflowWithSensitive(ctx context.Context, req app.CommonRequest, workflow app.Workflow, primary string, sensitive []string) (*app.WorkflowResponse, error) {
	resp, runCtx, err := s.prepare(ctx, req, workflow.Name(), readOnly(workflow, req))
	if err != nil {
		return resp, err
	}
	runCtx.SensitiveValues = compactSecrets(sensitive)
	if cancelled := s.cancelled(ctx, runCtx, resp); cancelled {
		return resp, nil
	}
	err = app.RunWorkflow(runCtx, workflow)
	if ctx.Err() != nil {
		s.markCancelled(runCtx, resp)
		return s.finish(runCtx, resp, primary, nil)
	}
	return s.finish(runCtx, resp, primary, err)
}

func (s *Service) prepare(ctx context.Context, req app.CommonRequest, command string, readOnly bool) (*app.WorkflowResponse, *app.AppContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	started := time.Now()
	runCtx := app.NewContext()
	runCtx.StartedAt = started
	runCtx.Run.StartedAt = started
	runCtx.Run.CommandName = command
	runCtx.Run.WorkflowName = command
	runCtx.Run.ReadOnly = readOnly
	runCtx.Run.DryRun = req.DryRun
	runCtx.Quiet = req.Quiet
	runCtx.NonInteractive = req.NonInteractive
	runCtx.Force = req.Force
	runCtx.JSONOutput = req.JSONOutput
	if req.Quiet {
		runCtx.Mode = app.RunModeQuiet
	}
	if req.NonInteractive {
		runCtx.Mode = app.RunModeCLI
	}
	runCtx.Progress = app.RedactingProgressSink{Next: s.Options.ProgressSink}

	effective, err := config.Load(config.LoadOptions{ExplicitPath: req.ConfigPath, ProfileOverride: req.Profile})
	if err != nil {
		runCtx.ExitCode = app.ExitInvalidInput
		return baseResponse(runCtx, command), runCtx, err
	}
	runCtx.Config = effective.Config
	runCtx.ConfigMeta = effective.Metadata()
	runCtx.ReportRoot = reportRoot(req, effective)
	if strings.TrimSpace(req.OutputDir) != "" {
		runCtx.OutputDir = req.OutputDir
	} else {
		runCtx.OutputDir = timestampedDir(runCtx.ReportRoot, started)
	}
	runCtx.Run.ReportDir = runCtx.OutputDir

	reporter, err := reports.New(runCtx.OutputDir)
	if err != nil {
		runCtx.ExitCode = app.ExitUnexpectedError
		return baseResponse(runCtx, command), runCtx, err
	}
	reporter.Run = &runCtx.Run
	reporter.Results = &runCtx.Results
	runCtx.Reporter = reporter
	_ = runCtx.Reporter.WriteJSON("config-metadata", runCtx.ConfigMeta)

	level := effective.Config.Logging.Level
	if strings.TrimSpace(req.LogLevel) != "" {
		if _, err := logging.ParseLevel(req.LogLevel); err != nil {
			runCtx.ExitCode = app.ExitInvalidInput
			return baseResponse(runCtx, command), runCtx, err
		}
		level = req.LogLevel
	}
	logger, err := logging.NewWithOptions(reports.LogPath(runCtx.OutputDir), logging.Options{
		Quiet:    true,
		Level:    level,
		RunID:    runCtx.Run.RunID,
		Workflow: command,
	})
	if err != nil {
		runCtx.ExitCode = app.ExitUnexpectedError
		return baseResponse(runCtx, command), runCtx, err
	}
	runCtx.Logger = logger
	runCtx.AddResult(app.OperationResult{
		Step:      "policy.profile",
		Target:    runCtx.Config.Profile,
		Status:    app.OperationStatusSuccess,
		Message:   "Active policy profile: " + runCtx.Config.Profile,
		Timestamp: time.Now(),
	})
	return baseResponse(runCtx, command), runCtx, nil
}

func (s *Service) cancelled(ctx context.Context, runCtx *app.AppContext, resp *app.WorkflowResponse) bool {
	if ctx == nil || ctx.Err() == nil {
		return false
	}
	s.markCancelled(runCtx, resp)
	_, _ = s.finish(runCtx, resp, "", nil)
	return true
}

func (s *Service) markCancelled(runCtx *app.AppContext, resp *app.WorkflowResponse) {
	runCtx.ExitCode = app.ExitInvalidInput
	runCtx.AddResult(app.OperationResult{
		Step:            "workflow.cancelled",
		Target:          runCtx.Run.WorkflowName,
		Status:          app.OperationStatusFailed,
		Message:         "Workflow cancelled",
		FailureCategory: "cancelled",
		Timestamp:       time.Now(),
	})
	resp.Meta.Status = string(app.WorkflowStatusCancelled)
	resp.Meta.ExitCode = app.ExitInvalidInput
	resp.Errors = append(resp.Errors, "workflow cancelled")
}

func (s *Service) finish(runCtx *app.AppContext, resp *app.WorkflowResponse, primary string, err error) (*app.WorkflowResponse, error) {
	priorStatus := resp.Meta.Status
	if runCtx.Logger != nil {
		_ = runCtx.Logger.Close()
	}
	runCtx.FinishRun()
	if runCtx.Reporter != nil {
		_ = runCtx.Reporter.WriteOperations(runCtx.Results)
		_ = ensureSummaryFile(runCtx)
	}
	resp.Meta = metaFromContext(runCtx, primary)
	if priorStatus == string(app.WorkflowStatusCancelled) {
		resp.Meta.Status = priorStatus
	} else if resp.Meta.Status == "" || resp.Meta.Status == string(app.WorkflowStatusSuccess) {
		resp.Meta.Status = workflowStatus(runCtx.ExitCode, err)
	}
	resp.Result = runCtx.JSONValue
	resp.Timeline = append([]app.OperationResult{}, runCtx.Results...)
	resp.Policy = runCtx.ConfigPolicy()
	resp.Warnings = append(resp.Warnings, stringSliceField(runCtx.JSONValue, "Warnings")...)
	resp.Errors = append(resp.Errors, stringSliceField(runCtx.JSONValue, "Errors")...)
	if err != nil {
		resp.Errors = append(resp.Errors, runCtx.RedactString(err.Error()))
	}
	redactResponse(runCtx, resp)
	return resp, sanitizeErr(runCtx, err)
}

func baseResponse(runCtx *app.AppContext, command string) *app.WorkflowResponse {
	resp := &app.WorkflowResponse{Meta: metaFromContext(runCtx, ""), Policy: runCtx.ConfigPolicy()}
	resp.Meta.Command = command
	resp.Meta.Workflow = command
	resp.Meta.Status = string(app.WorkflowStatusSuccess)
	return resp
}

func metaFromContext(ctx *app.AppContext, primary string) app.WorkflowResponseMeta {
	finished := ctx.Run.FinishedAt
	if finished.IsZero() {
		finished = time.Now()
	}
	return app.WorkflowResponseMeta{
		RunID:             ctx.Run.RunID,
		Workflow:          ctx.Run.WorkflowName,
		Command:           ctx.Run.CommandName,
		Status:            workflowStatus(ctx.ExitCode, nil),
		ExitCode:          ctx.ExitCode,
		ReportDir:         ctx.OutputDir,
		SummaryFile:       optionalFile(ctx.OutputDir, "summary.txt"),
		OperationsFile:    optionalFile(ctx.OutputDir, "operations.json"),
		PrimaryResultFile: optionalFile(ctx.OutputDir, primary),
		StartedAt:         ctx.Run.StartedAt,
		FinishedAt:        finished,
		DurationMS:        finished.Sub(ctx.Run.StartedAt).Milliseconds(),
	}
}

func workflowStatus(exitCode int, err error) string {
	if errors.Is(err, context.Canceled) {
		return string(app.WorkflowStatusCancelled)
	}
	if err != nil || exitCode >= app.ExitInvalidInput {
		return string(app.WorkflowStatusFailed)
	}
	if exitCode == app.ExitWarnings {
		return string(app.WorkflowStatusWarning)
	}
	return string(app.WorkflowStatusSuccess)
}

func optionalFile(dir string, name string) string {
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(name) == "" {
		return ""
	}
	return filepath.Join(dir, name)
}

func reportRoot(req app.CommonRequest, effective config.EffectiveConfig) string {
	if strings.TrimSpace(req.OutputDir) != "" {
		return req.OutputDir
	}
	if strings.TrimSpace(effective.Config.Reports.Root) != "" {
		return effective.Config.Reports.Root
	}
	return config.DefaultReportRoot
}

func timestampedDir(root string, startedAt time.Time) string {
	base := reports.TimestampedDir(root, startedAt)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 1; i < 100; i++ {
		candidate := strings.TrimRight(base, `\/`) + "-" + twoDigits(i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return base
}

func twoDigits(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func readOnly(workflow app.Workflow, req app.CommonRequest) bool {
	name := strings.ToLower(workflow.Name())
	if req.DryRun || strings.Contains(name, "dry-run") {
		return true
	}
	return name == "check" || name == "verify" || name == "preflight" || name == "collect-report"
}

func stringSliceField(value any, field string) []string {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil
	}
	raw, ok := decoded[strings.ToLower(field)]
	if !ok {
		raw = decoded[field]
	}
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func compactSecrets(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func redactResponse(ctx *app.AppContext, resp *app.WorkflowResponse) {
	if ctx == nil || resp == nil {
		return
	}
	resp.Meta.ReportDir = ctx.RedactString(resp.Meta.ReportDir)
	resp.Meta.SummaryFile = ctx.RedactString(resp.Meta.SummaryFile)
	resp.Meta.OperationsFile = ctx.RedactString(resp.Meta.OperationsFile)
	resp.Meta.PrimaryResultFile = ctx.RedactString(resp.Meta.PrimaryResultFile)
	resp.Warnings = redactStrings(ctx, resp.Warnings)
	resp.Errors = redactStrings(ctx, resp.Errors)
	for i := range resp.Timeline {
		resp.Timeline[i] = ctx.RedactOperationResult(resp.Timeline[i])
	}
	resp.Result = redactValue(ctx, resp.Result)
}

func redactStrings(ctx *app.AppContext, values []string) []string {
	for i := range values {
		values[i] = ctx.RedactString(values[i])
	}
	return values
}

func redactValue(ctx *app.AppContext, value any) any {
	if value == nil {
		return nil
	}
	if len(ctx.SensitiveValues) == 0 {
		return value
	}
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	redacted := ctx.RedactString(string(data))
	var decoded any
	if err := json.Unmarshal([]byte(redacted), &decoded); err != nil {
		return value
	}
	return decoded
}

func sanitizeErr(ctx *app.AppContext, err error) error {
	if err == nil {
		return nil
	}
	return errors.New(ctx.RedactString(err.Error()))
}

func ensureSummaryFile(ctx *app.AppContext) error {
	if ctx == nil || ctx.Reporter == nil || strings.TrimSpace(ctx.OutputDir) == "" {
		return nil
	}
	path := filepath.Join(ctx.OutputDir, "summary.txt")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	status := workflowStatus(ctx.ExitCode, nil)
	var b strings.Builder
	b.WriteString("Kigrepair Workflow Summary\n\n")
	b.WriteString("Workflow: " + ctx.Run.WorkflowName + "\n")
	b.WriteString("Status: " + status + "\n")
	b.WriteString("Exit code: " + strconv.Itoa(ctx.ExitCode) + "\n")
	b.WriteString("Report directory: " + ctx.OutputDir + "\n")
	if len(ctx.Results) > 0 {
		b.WriteString("\n")
		b.WriteString(reports.FormatOperationTimeline(ctx.Results))
	}
	return ctx.Reporter.WriteText("summary", ctx.RedactString(b.String()))
}

var _ app.WorkflowService = (*Service)(nil)
