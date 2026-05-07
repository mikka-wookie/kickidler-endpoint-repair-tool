//go:build windows && cgo && !oldgui

package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kigapp "kigrepair/internal/app"
	"kigrepair/internal/app/workflowservice"
	"kigrepair/internal/config"
	"kigrepair/internal/safety"
	"kigrepair/internal/version"
	"kigrepair/internal/winapi"
)

type fyneGUI struct {
	app        fyne.App
	window     fyne.Window
	guard      *singleInstanceGuard
	controller *Controller

	mu          sync.Mutex
	running     bool
	relaunching bool
	cancel      context.CancelFunc

	admin         bool
	activeProfile string
	reportRoot    string
	latest        ResultView

	headerTitle   *widget.Label
	headerMeta    *widget.Label
	adminBadge    *widget.Label
	workflowBadge *widget.Label
	statusBadge   *widget.Label
	profileSelect *widget.Select

	systemStatus *widget.Label
	timeline     *widget.Label
	details      *widget.Label

	installerEntry *widget.Entry
	inviteEntry    *widget.Entry

	buttons       []*widget.Button
	cancelButton  *widget.Button
	restartButton *widget.Button
	openSummary   *widget.Button
	openOps       *widget.Button
	copyReport    *widget.Button
	copyBundle    *widget.Button
}

func runGUI() error {
	opts := parseGUIOptions(os.Args[1:])
	guard, err := acquireSingleInstance(opts.AllowMultiple, opts.ElevatedChild)
	if err != nil {
		return err
	}

	a := app.NewWithID("kigrepair.gui")
	w := a.NewWindow("kigrepair")
	w.Resize(fyne.NewSize(guiMinWidth, guiMinHeight))
	w.SetMaster()

	gui := &fyneGUI{
		app:           a,
		window:        w,
		guard:         guard,
		activeProfile: "standard",
		reportRoot:    config.DefaultReportRoot,
		latest:        InitialResultView(),
	}
	gui.controller = NewController(workflowservice.New(kigapp.WorkflowOptions{ProgressSink: guiProgressSink{gui: gui}}), gui.confirmExactYES)
	gui.loadLightweightConfig(opts)
	gui.build()
	gui.refresh(InitialResultView())
	gui.refreshRunning(false)
	w.SetCloseIntercept(func() {
		gui.clearInvite()
		gui.controller.Cancel()
		if gui.guard != nil {
			gui.guard.Release()
		}
		a.Quit()
	})
	go gui.detectAdmin()
	w.ShowAndRun()
	if gui.guard != nil {
		gui.guard.Release()
	}
	return nil
}

func (g *fyneGUI) loadLightweightConfig(opts GUIOptions) {
	effective, err := config.Load(config.LoadOptions{})
	if err == nil {
		if profile := strings.TrimSpace(effective.Config.Profile); profile != "" {
			g.activeProfile = profile
		}
		if root := strings.TrimSpace(effective.Config.Reports.Root); root != "" {
			g.reportRoot = root
		}
	}
	if profile := strings.TrimSpace(opts.Profile); profile != "" {
		g.activeProfile = profile
	}
}

func (g *fyneGUI) build() {
	info := version.Get()
	g.headerTitle = widget.NewLabel("kigrepair")
	g.headerTitle.TextStyle = fyne.TextStyle{Bold: true}
	g.headerMeta = widget.NewLabel(fmt.Sprintf("Kickidler Grabber Repair Utility    %s / %s", emptyAs(info.Version, "dev"), emptyAs(info.Commit, "dev")))
	g.adminBadge = widget.NewLabel("Limited mode")
	g.workflowBadge = widget.NewLabel("Workflow: none")
	g.statusBadge = widget.NewLabel("Status: idle")
	g.profileSelect = widget.NewSelect(profileOptions(g.activeProfile), func(value string) {
		if strings.TrimSpace(value) != "" {
			g.activeProfile = value
		}
	})
	g.profileSelect.SetSelected(g.activeProfile)
	header := container.NewBorder(nil, nil,
		container.NewVBox(g.headerTitle, g.headerMeta),
		container.NewVBox(g.profileSelect, g.adminBadge, g.workflowBadge, g.statusBadge),
		widget.NewSeparator(),
	)

	g.systemStatus = wrapLabel("No workflow has been run yet. Start with Check or Verify.")
	systemCard := widget.NewCard("System Status", "", container.NewMax(g.systemStatus))

	check := g.actionButton("Check", theme.SearchIcon(), ActionCheck)
	verify := g.actionButton("Verify", theme.ConfirmIcon(), ActionVerify)
	collect := g.actionButton("Collect Bundle", theme.FolderNewIcon(), ActionCollectReport)
	openReports := widget.NewButtonWithIcon("Open Reports Folder", theme.FolderOpenIcon(), func() {
		target := g.latest.ReportDir
		if strings.TrimSpace(target) == "" {
			target = g.reportRoot
		}
		g.openPathAsync(target, "Reports folder is not available yet. Run Check first.")
	})
	g.restartButton = widget.NewButtonWithIcon("Restart as Administrator", theme.LoginIcon(), g.restartAsAdmin)
	g.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), g.cancelWorkflow)
	quick := widget.NewCard("Quick Actions", "", container.NewHBox(check, verify, collect, openReports, g.restartButton, g.cancelButton))

	g.installerEntry = widget.NewEntry()
	g.installerEntry.SetPlaceHolder(`C:\Path\grabber.msi`)
	browse := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), g.browseInstaller)
	g.inviteEntry = widget.NewPasswordEntry()
	g.inviteEntry.SetPlaceHolder("Invite")
	preflight := g.actionButton("Preflight", theme.InfoIcon(), ActionPreflight)
	dryRun := g.actionButton("Repair Dry-Run", theme.VisibilityIcon(), ActionRepairDryRun)
	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Installer path", container.NewBorder(nil, nil, nil, browse, g.installerEntry)),
			widget.NewFormItem("Invite", g.inviteEntry),
		),
		container.NewHBox(preflight, dryRun),
	)
	repairCard := widget.NewCard("Repair Preparation", "", form)

	dangerText := wrapLabel("Real repair can modify services, Defender exclusions, MSI state, and validated leftovers.")
	realRepair := widget.NewButtonWithIcon("RUN REAL REPAIR", theme.WarningIcon(), func() {
		if !g.admin {
			g.showError("Real repair requires Administrator rights.")
			return
		}
		g.runWorkflow(ActionRepair)
	})
	realRepair.Importance = widget.HighImportance
	g.buttons = append(g.buttons, realRepair)
	dangerCard := widget.NewCard("Danger Zone", "", container.NewVBox(dangerText, realRepair))
	prepSplit := container.NewHSplit(repairCard, dangerCard)
	prepSplit.SetOffset(0.66)

	g.timeline = wrapLabel(FormatTimeline(g.latest.Timeline))
	timelineCard := widget.NewCard("Timeline", "", container.NewVScroll(g.timeline))

	g.details = wrapLabel("")
	g.openSummary = widget.NewButtonWithIcon("Open Summary", theme.DocumentIcon(), func() {
		g.openPathAsync(firstNonEmptyString(g.latest.SummaryFile, filepathFromReport(g.latest.ReportDir, "summary.txt")), "summary.txt is not available for the latest run.")
	})
	g.openOps = widget.NewButtonWithIcon("Open Operations", theme.DocumentIcon(), func() {
		g.openPathAsync(firstNonEmptyString(g.latest.OperationsFile, filepathFromReport(g.latest.ReportDir, "operations.json")), "operations.json is not available for the latest run.")
	})
	g.copyReport = widget.NewButtonWithIcon("Copy Report Path", theme.ContentCopyIcon(), func() { g.copyPath(g.latest.ReportDir, "Report path is not available yet.") })
	g.copyBundle = widget.NewButtonWithIcon("Copy Bundle Path", theme.ContentCopyIcon(), func() { g.copyPath(g.latest.SupportBundlePath, "Support bundle path is not available yet.") })
	reportsList := g.actionButton("Reports List", theme.ViewRefreshIcon(), ActionReportsList)
	cleanupDry := g.actionButton("Cleanup Dry-Run", theme.VisibilityIcon(), ActionReportsCleanupDry)
	detailsCard := widget.NewCard("Details", "", container.NewBorder(nil,
		container.NewVBox(
			container.NewHBox(g.openSummary, g.openOps),
			container.NewHBox(g.copyReport, g.copyBundle, reportsList, cleanupDry),
		),
		nil, nil,
		container.NewVScroll(g.details),
	))
	bottomSplit := container.NewHSplit(timelineCard, detailsCard)
	bottomSplit.SetOffset(0.62)
	content := container.NewVSplit(container.NewVBox(systemCard, quick, prepSplit), bottomSplit)
	content.SetOffset(0.48)

	g.window.SetContent(container.NewBorder(header, widget.NewLabel("Ready"), nil, nil, content))
}

func (g *fyneGUI) actionButton(label string, icon fyne.Resource, action Action) *widget.Button {
	button := widget.NewButtonWithIcon(label, icon, func() { g.runWorkflow(action) })
	g.buttons = append(g.buttons, button)
	return button
}

func (g *fyneGUI) runWorkflow(action Action) {
	if g.isRunning() {
		g.showError("Another workflow is already running.")
		return
	}
	inputs := Inputs{
		InstallerPath: g.installerEntry.Text,
		InviteValue:   g.inviteEntry.Text,
		Profile:       g.activeProfile,
	}
	if needsInstaller(action) && strings.TrimSpace(inputs.InstallerPath) == "" {
		g.localNotReady("Installer file was not found.")
		if action != ActionRepair {
			g.clearInvite()
		}
		return
	}
	if action == ActionRepair && strings.TrimSpace(inputs.InviteValue) == "" {
		g.localNotReady("Invite is required for this workflow.")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	g.setCancel(cancel)
	g.refresh(ResultView{Status: "Running", Workflow: string(action), Timeline: []TimelineItem{{Step: "workflow-started", Status: "success", Message: supportActionStarted(action)}}})
	g.refreshRunning(true)
	go g.progressPump(ctx)
	go func() {
		view, err := g.controller.Run(ctx, action, inputs)
		if err != nil && strings.TrimSpace(err.Error()) != "" {
			view.Errors = append(view.Errors, safety.RedactString(err.Error()))
		}
		view.Timeline = capTimeline(SupportTimeline(view), 25)
		fyne.Do(func() {
			g.clearInvite()
			g.refresh(view)
			g.setCancel(nil)
			g.refreshRunning(false)
		})
	}()
}

func (g *fyneGUI) progressPump(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !g.controller.IsRunning() {
				return
			}
			items := capTimeline(SupportTimeline(ResultView{Timeline: g.controller.ProgressTimeline()}), 25)
			fyne.Do(func() {
				g.timeline.SetText(FormatTimeline(items))
			})
		}
	}
}

func (g *fyneGUI) cancelWorkflow() {
	g.clearInvite()
	g.controller.Cancel()
	g.mu.Lock()
	if g.cancel != nil {
		g.cancel()
	}
	g.mu.Unlock()
}

func (g *fyneGUI) setCancel(cancel context.CancelFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cancel = cancel
	g.running = cancel != nil
}

func (g *fyneGUI) isRunning() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.running || g.controller.IsRunning()
}

func (g *fyneGUI) refresh(view ResultView) {
	if strings.TrimSpace(view.Status) == "" {
		view.Status = "Idle"
	}
	g.latest = view
	if view.ReportDir != "" {
		g.latest.ReportDir = view.ReportDir
	}
	g.workflowBadge.SetText("Workflow: " + emptyAs(view.Workflow, "none"))
	g.statusBadge.SetText("Status: " + strings.ToLower(emptyAs(view.Status, "idle")))
	g.systemStatus.SetText(FormatSystemStatus(view))
	g.timeline.SetText(FormatTimeline(capTimeline(view.Timeline, 25)))
	g.details.SetText(g.detailsText(view))
	g.refreshFileButtons(view)
}

func (g *fyneGUI) refreshRunning(running bool) {
	g.mu.Lock()
	g.running = running
	g.mu.Unlock()
	state := ComputeButtonState(running)
	for _, button := range g.buttons {
		if state.ActionsEnabled {
			button.Enable()
		} else {
			button.Disable()
		}
	}
	if state.CancelEnabled {
		g.cancelButton.Enable()
	} else {
		g.cancelButton.Disable()
	}
	if g.admin || g.relaunching || running {
		g.restartButton.Disable()
	} else {
		g.restartButton.Enable()
	}
	g.refreshFileButtons(g.latest)
}

func (g *fyneGUI) refreshFileButtons(view ResultView) {
	hasReport := strings.TrimSpace(view.ReportDir) != ""
	hasSummary := strings.TrimSpace(firstNonEmptyString(view.SummaryFile, filepathFromReport(view.ReportDir, "summary.txt"))) != ""
	hasOps := strings.TrimSpace(firstNonEmptyString(view.OperationsFile, filepathFromReport(view.ReportDir, "operations.json"))) != ""
	hasBundle := strings.TrimSpace(view.SupportBundlePath) != ""
	setButtonEnabled(g.openSummary, hasReport && hasSummary)
	setButtonEnabled(g.openOps, hasReport && hasOps)
	setButtonEnabled(g.copyReport, hasReport)
	setButtonEnabled(g.copyBundle, hasBundle)
}

func (g *fyneGUI) detailsText(view ResultView) string {
	return safety.RedactString(strings.Join([]string{
		"Report directory: " + emptyAs(view.ReportDir, "not available"),
		"Summary: " + emptyAs(firstNonEmptyString(view.SummaryFile, filepathFromReport(view.ReportDir, "summary.txt")), "not available"),
		"Operations: " + emptyAs(firstNonEmptyString(view.OperationsFile, filepathFromReport(view.ReportDir, "operations.json")), "not available"),
		"Primary result: " + emptyAs(view.PrimaryResultFile, "not available"),
		"Cleanup plan: " + emptyAs(filepathFromReport(view.ReportDir, "cleanup-plan.json"), "not available"),
		"Support bundle: " + emptyAs(view.SupportBundlePath, "not available"),
	}, "\n"))
}

func (g *fyneGUI) confirmExactYES(ctx context.Context, req ConfirmationRequest) (bool, error) {
	result := make(chan bool, 1)
	fyne.Do(func() {
		entry := widget.NewEntry()
		entry.SetPlaceHolder(req.RequiredText)
		content := container.NewVBox(wrapLabel(req.Message), entry)
		dialog.NewCustomConfirm(req.Title, "Continue", "Cancel", content, func(ok bool) {
			result <- ok && AcceptExactYES(entry.Text)
		}, g.window).Show()
	})
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case ok := <-result:
		return ok, nil
	}
}

func (g *fyneGUI) browseInstaller() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			g.showError(err.Error())
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		path := reader.URI().Path()
		g.installerEntry.SetText(path)
		if warning := installerFilenameWarning(path); warning != "" {
			g.showError(warning)
		}
	}, g.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".msi"}))
	fd.Show()
}

func (g *fyneGUI) openPathAsync(path string, missing string) {
	go func() {
		if err := OpenPath(path); err != nil {
			fyne.Do(func() { g.showError(missing) })
		}
	}()
}

func (g *fyneGUI) copyPath(path string, missing string) {
	if strings.TrimSpace(path) == "" {
		g.showError(missing)
		return
	}
	g.app.Clipboard().SetContent(safety.RedactString(path))
	dialog.ShowInformation("kigrepair", "Path copied.", g.window)
}

func (g *fyneGUI) restartAsAdmin() {
	if g.admin || g.relaunching || g.isRunning() {
		return
	}
	dialog.ShowConfirm("kigrepair", "Restart kigrepair as Administrator?\n\nCurrent window will close.", func(ok bool) {
		if !ok {
			return
		}
		g.relaunching = true
		g.clearInvite()
		g.refreshRunning(false)
		go func() {
			args := []string{"--elevated-child"}
			if strings.TrimSpace(g.activeProfile) != "" {
				args = append(args, "--profile", g.activeProfile)
			}
			if err := winapi.RelaunchElevated(args); err != nil {
				fyne.Do(func() {
					g.relaunching = false
					g.refreshRunning(false)
					g.showError(err.Error())
				})
				return
			}
			fyne.Do(func() {
				if g.guard != nil {
					g.guard.Release()
				}
				g.window.Close()
			})
		}()
	}, g.window)
}

func (g *fyneGUI) detectAdmin() {
	admin := winapi.IsAdmin()
	fyne.Do(func() {
		g.admin = admin
		if admin {
			g.adminBadge.SetText("Administrator")
		} else {
			g.adminBadge.SetText("Limited mode: Administrator rights are required for full service, Defender, and repair checks.")
		}
		g.refreshRunning(g.isRunning())
	})
}

func (g *fyneGUI) localNotReady(message string) {
	view := g.latest
	view.Status = "Not ready"
	view.BlockingReasons = dedupeStrings(append(view.BlockingReasons, message))
	view.Timeline = append(view.Timeline, TimelineItem{Step: "not-ready", Status: "warning", Message: message})
	g.refresh(view)
}

func (g *fyneGUI) clearInvite() {
	if g.inviteEntry != nil {
		g.inviteEntry.SetText("")
	}
}

func (g *fyneGUI) showError(message string) {
	dialog.ShowError(fmt.Errorf("%s", safety.RedactString(message)), g.window)
}

type guiProgressSink struct {
	gui *fyneGUI
}

func (s guiProgressSink) OnProgress(event kigapp.ProgressEvent) {
	if s.gui != nil && s.gui.controller != nil {
		s.gui.controller.appendProgress(event)
	}
}

func MinWindowSize() fyne.Size {
	return fyne.NewSize(guiMinWidth, guiMinHeight)
}

func profileOptions(active string) []string {
	options := []string{"standard", "conservative", "diagnostic"}
	active = strings.TrimSpace(active)
	if active == "" {
		return options
	}
	for _, option := range options {
		if option == active {
			return options
		}
	}
	return append([]string{active}, options...)
}

func wrapLabel(text string) *widget.Label {
	label := widget.NewLabel(safety.RedactString(text))
	label.Wrapping = fyne.TextWrapWord
	return label
}

func setButtonEnabled(button *widget.Button, enabled bool) {
	if button == nil {
		return
	}
	if enabled {
		button.Enable()
	} else {
		button.Disable()
	}
}

func needsInstaller(action Action) bool {
	return action == ActionPreflight || action == ActionRepairDryRun || action == ActionRepair
}

func supportActionStarted(action Action) string {
	switch action {
	case ActionCheck:
		return "Check started"
	case ActionVerify:
		return "Verify started"
	case ActionCollectReport:
		return "Collect report started"
	case ActionPreflight:
		return "Preflight started"
	case ActionRepairDryRun:
		return "Repair dry-run started"
	case ActionRepair:
		return "Real repair started"
	default:
		return string(action) + " started"
	}
}

func capTimeline(items []TimelineItem, limit int) []TimelineItem {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	out := append([]TimelineItem(nil), items[:limit-1]...)
	out = append(out, TimelineItem{Step: "timeline-summary", Status: "info", Message: fmt.Sprintf("%d additional events saved in report files.", len(items)-len(out))})
	return out
}

func installerFilenameWarning(path string) string {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(path)))
	if name == "" {
		return ""
	}
	switch name {
	case "grabberem.x64.msi", "grabberem.x32.msi", "grabbertt.x64.msi", "grabbertt.x32.msi", "grabber.msi":
		return ""
	default:
		return "Unsupported installer filename. Backend validation may reject it."
	}
}

func filepathFromReport(reportDir string, name string) string {
	if strings.TrimSpace(reportDir) == "" {
		return ""
	}
	return filepath.Join(reportDir, name)
}
