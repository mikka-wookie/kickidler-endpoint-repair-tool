package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/safety"
)

type Action string

const (
	ActionCheck             Action = "check"
	ActionVerify            Action = "verify"
	ActionPreflight         Action = "preflight"
	ActionRepairDryRun      Action = "repair_dry_run"
	ActionRepair            Action = "repair"
	ActionCollectReport     Action = "collect_report"
	ActionReportsList       Action = "reports_list"
	ActionReportsCleanupDry Action = "reports_cleanup_dry_run"
)

type Inputs struct {
	InstallerPath string
	InviteValue   string
	Profile       string
	ConfigPath    string
	OutputDir     string
}

type ConfirmationFunc func(context.Context, ConfirmationRequest) (bool, error)

type ConfirmationRequest struct {
	Title        string
	Message      string
	RequiredText string
}

type ResultView struct {
	RunID                 string
	Status                string
	Workflow              string
	ReportDir             string
	SummaryFile           string
	OperationsFile        string
	PrimaryResultFile     string
	SupportBundlePath     string
	Classification        string
	Recommendation        string
	PrimaryIssueCode      string
	NextRecommendedAction string
	Warnings              []string
	Errors                []string
	Timeline              []TimelineItem
	Reports               []string
}

type TimelineItem struct {
	Time    time.Time
	Stage   string
	Step    string
	Target  string
	Status  string
	Message string
}

type Controller struct {
	service app.WorkflowService
	confirm ConfirmationFunc

	mu      sync.Mutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	events  []app.ProgressEvent
	latest  ResultView
}

func NewController(service app.WorkflowService, confirm ConfirmationFunc) *Controller {
	if confirm == nil {
		confirm = func(context.Context, ConfirmationRequest) (bool, error) { return false, nil }
	}
	return &Controller{service: service, confirm: confirm}
}

func (c *Controller) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *Controller) Latest() ResultView {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneResultView(c.latest)
}

func (c *Controller) Cancel() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Controller) ProgressSink() app.ProgressSink {
	return progressSink{controller: c}
}

func (c *Controller) Run(parent context.Context, action Action, inputs Inputs) (ResultView, error) {
	if c.service == nil {
		return ResultView{}, errors.New("workflow service is not configured")
	}
	if parent == nil {
		parent = context.Background()
	}
	if !c.begin(parent) {
		return ResultView{}, errors.New("another workflow is already running")
	}
	defer c.end()
	defer c.clearInvite(&inputs)

	common := app.CommonRequest{
		ConfigPath:     strings.TrimSpace(inputs.ConfigPath),
		Profile:        strings.TrimSpace(inputs.Profile),
		OutputDir:      strings.TrimSpace(inputs.OutputDir),
		Quiet:          true,
		NonInteractive: true,
		JSONOutput:     false,
		LogLevel:       "",
	}

	ctx := c.context()
	var resp *app.WorkflowResponse
	var err error
	switch action {
	case ActionCheck:
		resp, err = c.service.Check(ctx, app.CheckRequest{CommonRequest: common})
	case ActionVerify:
		resp, err = c.service.Verify(ctx, app.VerifyRequest{CommonRequest: common})
	case ActionPreflight:
		resp, err = c.service.Preflight(ctx, app.PreflightRequest{CommonRequest: common, InstallerRequestFields: installerFields(inputs)})
	case ActionRepairDryRun:
		common.DryRun = true
		resp, err = c.service.RepairPlan(ctx, app.RepairPlanRequest{CommonRequest: common, InstallerRequestFields: installerFields(inputs)})
	case ActionRepair:
		if err := requireRealRepairInputs(inputs); err != nil {
			resp = failedViewResponse("repair", err)
			break
		}
		accepted, confirmErr := c.confirm(ctx, RealRepairConfirmation())
		if confirmErr != nil {
			err = sanitizeError(confirmErr)
			resp = failedViewResponse("repair", err)
			break
		}
		if !accepted {
			err = errors.New("repair confirmation was not accepted")
			resp = failedViewResponse("repair", err)
			break
		}
		common.Yes = true
		resp, err = c.service.Repair(ctx, app.RepairRequest{CommonRequest: common, InstallerRequestFields: installerFields(inputs)})
	case ActionCollectReport:
		resp, err = c.service.CollectReport(ctx, app.CollectReportRequest{CommonRequest: common, IncludeEventLogs: true, IncludeHistory: true, HistoryLimit: 5})
	case ActionReportsList:
		resp, err = c.service.ReportsList(ctx, app.ReportsListRequest{CommonRequest: common, Limit: 20})
	case ActionReportsCleanupDry:
		common.DryRun = true
		resp, err = c.service.ReportsCleanupPlan(ctx, app.ReportsCleanupPlanRequest{CommonRequest: common})
	default:
		err = fmt.Errorf("unknown GUI action: %s", action)
		resp = failedViewResponse(string(action), err)
	}

	view := BuildResultView(resp)
	view.Timeline = append(c.progressTimeline(), view.Timeline...)
	if err != nil {
		view.Errors = append(view.Errors, sanitizeError(err).Error())
	}
	c.mu.Lock()
	c.latest = cloneResultView(view)
	c.mu.Unlock()
	return view, sanitizeError(err)
}

func (c *Controller) begin(parent context.Context) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return false
	}
	ctx, cancel := context.WithCancel(parent)
	c.running = true
	c.ctx = ctx
	c.cancel = cancel
	c.events = nil
	c.latest = ResultView{Status: "running"}
	_ = ctx
	return true
}

func (c *Controller) context() context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *Controller) end() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx = nil
	c.cancel = nil
}

func (c *Controller) appendProgress(event app.ProgressEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, app.RedactProgressEvent(event))
}

func (c *Controller) progressTimeline() []TimelineItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := make([]TimelineItem, 0, len(c.events))
	for _, event := range c.events {
		when := event.StartedAt
		if when.IsZero() {
			when = event.FinishedAt
		}
		items = append(items, TimelineItem{
			Time:    when,
			Stage:   event.Stage,
			Step:    event.OperationID,
			Target:  event.ResultFile,
			Status:  string(event.Status),
			Message: event.Message,
		})
	}
	return items
}

func (c *Controller) clearInvite(inputs *Inputs) {
	if inputs != nil {
		inputs.InviteValue = ""
	}
}

func installerFields(inputs Inputs) app.InstallerRequestFields {
	return app.InstallerRequestFields{
		InstallerPath: strings.TrimSpace(inputs.InstallerPath),
		InviteValue:   strings.TrimSpace(inputs.InviteValue),
	}
}

func requireRealRepairInputs(inputs Inputs) error {
	if strings.TrimSpace(inputs.InstallerPath) == "" {
		return errors.New("installer path is required for real repair")
	}
	if strings.TrimSpace(inputs.InviteValue) == "" {
		return errors.New("invite is required for real repair")
	}
	return nil
}

func RealRepairConfirmation() ConfirmationRequest {
	return ConfirmationRequest{
		Title:        "Confirm Real Repair",
		RequiredText: "YES",
		Message: strings.Join([]string{
			"This will modify the system.",
			"It may stop/delete validated Grabber services/processes.",
			"It may uninstall/reinstall MSI.",
			"It may add Defender exclusions.",
			"A rollback/change snapshot will be written before changes.",
			"Type exact YES to continue.",
		}, "\r\n"),
	}
}

func AcceptExactYES(input string) bool {
	return input == "YES"
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(safety.RedactString(err.Error()))
}

func failedViewResponse(workflow string, err error) *app.WorkflowResponse {
	return &app.WorkflowResponse{
		Meta:   app.WorkflowResponseMeta{Workflow: workflow, Command: workflow, Status: string(app.WorkflowStatusFailed), FinishedAt: time.Now()},
		Errors: []string{safety.RedactString(err.Error())},
	}
}

type progressSink struct {
	controller *Controller
}

func (s progressSink) OnProgress(event app.ProgressEvent) {
	if s.controller != nil {
		s.controller.appendProgress(event)
	}
}

func OpenPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("path is empty")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return openPath(path)
}

func cloneResultView(in ResultView) ResultView {
	out := in
	out.Warnings = append([]string(nil), in.Warnings...)
	out.Errors = append([]string(nil), in.Errors...)
	out.Timeline = append([]TimelineItem(nil), in.Timeline...)
	out.Reports = append([]string(nil), in.Reports...)
	return out
}
