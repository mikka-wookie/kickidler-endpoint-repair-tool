//go:build windows && oldgui

package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"kigrepair/internal/app"
	"kigrepair/internal/app/workflowservice"
	"kigrepair/internal/config"
	"kigrepair/internal/safety"
	"kigrepair/internal/version"
	"kigrepair/internal/winapi"
)

const (
	wmCommand       = 0x0111
	wmClose         = 0x0010
	wmDestroy       = 0x0002
	wmSize          = 0x0005
	wmGetMinMaxInfo = 0x0024
	wmTimer         = 0x0113
	wmAppUI         = 0x8002

	bsPushButton  = 0x00000000
	bsGroupBox    = 0x00000007
	esAutoHScroll = 0x0080
	esMultiLine   = 0x0004
	esReadOnly    = 0x0800
	esPassword    = 0x0020
	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsThickFrame  = 0x00040000
	wsMinimizeBox = 0x00020000
	wsMaximizeBox = 0x00010000
	wsVisible     = 0x10000000
	wsChild       = 0x40000000
	wsBorder      = 0x00800000
	wsHScroll     = 0x00100000
	wsVScroll     = 0x00200000
	wsTabStop     = 0x00010000
	wsGroup       = 0x00020000

	swShow      = 5
	swHide      = 0
	mbYesNo     = 0x00000004
	idYes       = 6
	colorWindow = 5
)

const (
	idCheck = 1001 + iota
	idVerify
	idCollect
	idReportsList
	idOpenReports
	idBrowseInstaller
	idPreflight
	idRepairDryRun
	idRepair
	idCancel
	idOpenSelectedReport
	idReportsCleanupDry
	idRestartAdmin
	idInstaller
	idInvite
	idProfile
	idTimeline
	idHeader
	idSystem
	idSummary
	idReports
	idCopyReportPath
	idOpenOperations
	idCopyBundlePath
	idGroupSystem
	idGroupQuick
	idGroupRepair
	idGroupDanger
	idGroupTimeline
	idGroupDetails
	idStaticInstaller
	idStaticInvite
	idStaticProfile
	idStaticProfiles
	idStaticDanger1
	idStaticDanger2
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	comdlg32             = windows.NewLazySystemDLL("comdlg32.dll")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMsg     = user32.NewProc("TranslateMessage")
	procDispatchMsg      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procSetWindowText    = user32.NewProc("SetWindowTextW")
	procGetWindowText    = user32.NewProc("GetWindowTextW")
	procEnableWindow     = user32.NewProc("EnableWindow")
	procMessageBox       = user32.NewProc("MessageBoxW")
	procSendMessage      = user32.NewProc("SendMessageW")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")
	procPostMessage      = user32.NewProc("PostMessageW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procOutputDebug      = kernel32.NewProc("OutputDebugStringW")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGetOpenFileName  = comdlg32.NewProc("GetOpenFileNameW")
)

type mainWindow struct {
	hwnd            windows.Handle
	controller      *Controller
	controls        map[int]windows.Handle
	uiMu            sync.Mutex
	uiQueue         []func()
	latestReport    string
	latestSummary   string
	reportRoot      string
	configSource    string
	activeProfile   string
	admin           bool
	startedAt       time.Time
	relaunching     bool
	elevatedChild   bool
	currentWorkflow string
}

func runGUI() error {
	opts := parseGUIOptions(os.Args[1:])
	guard, err := acquireSingleInstance(opts.AllowMultiple, opts.ElevatedChild)
	if err != nil {
		messageBox(0, err.Error(), "kigrepair GUI")
		return err
	}
	defer guard.Release()

	w := &mainWindow{controls: map[int]windows.Handle{}, startedAt: time.Now(), elevatedChild: opts.ElevatedChild}
	uiDebug("UI startup started")
	w.controller = NewController(nil, w.confirm)
	w.controller.service = workflowservice.New(app.WorkflowOptions{ProgressSink: w.controller.ProgressSink()})
	effective, err := config.Load(config.LoadOptions{})
	if err == nil {
		w.configSource = effective.Metadata().Path
		w.activeProfile = effective.Config.Profile
		w.reportRoot = effective.Config.Reports.Root
	}
	if strings.TrimSpace(opts.Profile) != "" {
		w.activeProfile = strings.TrimSpace(opts.Profile)
	}
	if strings.TrimSpace(w.reportRoot) == "" {
		w.reportRoot = config.DefaultReportRoot
	}
	return w.run()
}

func (w *mainWindow) run() error {
	instance, _, _ := procGetModuleHandle.Call(0)
	className := utf16Ptr("KigrepairGuiWindow")
	wndproc := syscall.NewCallback(w.windowProc)
	wc := wndClassEx{
		Size:       uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:    wndproc,
		Instance:   windows.Handle(instance),
		ClassName:  className,
		Background: windows.Handle(colorWindow + 1),
	}
	if ret, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", err)
	}
	hwnd, _, err := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("kigrepair GUI MVP"))),
		wsOverlapped|wsCaption|wsSysMenu|wsThickFrame|wsMinimizeBox|wsMaximizeBox|wsVisible,
		80, 60, minWindowWidth, minWindowHeight,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", err)
	}
	w.hwnd = windows.Handle(hwnd)
	w.createControls()
	w.layoutControls(minWindowWidth-16, minWindowHeight-39)
	w.updateDashboard(InitialResultView())
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	procSetTimer.Call(hwnd, 1, 350, 0)
	uiDebug(fmt.Sprintf("UI startup finished in %d ms", time.Since(w.startedAt).Milliseconds()))
	go w.detectAdminAsync()

	var msg msg
	for {
		ret, _, callErr := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) == -1 {
			return fmt.Errorf("GetMessageW failed: %w", callErr)
		}
		if ret == 0 {
			return nil
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (w *mainWindow) createControls() {
	w.controls[idHeader] = createEdit(w.hwnd, "", 16, 12, 1148, 82, esMultiLine|esReadOnly|wsVScroll)

	w.controls[idGroupSystem] = createGroup(w.hwnd, "System Status", 16, 104, 1148, 146)
	w.controls[idSystem] = createEdit(w.hwnd, "", 32, 130, 1116, 104, esMultiLine|esReadOnly|wsVScroll)

	w.controls[idGroupQuick] = createGroup(w.hwnd, "Quick Actions", 16, 260, 1148, 70)
	createButton(w, idCheck, "Check", 32, 286, 100, 30)
	createButton(w, idVerify, "Verify", 140, 286, 100, 30)
	createButton(w, idCollect, "Collect Bundle", 248, 286, 130, 30)
	createButton(w, idOpenReports, "Open Reports Folder", 386, 286, 170, 30)
	createButton(w, idRestartAdmin, "Restart as Administrator", 564, 286, 190, 30)
	createButton(w, idCancel, "Cancel", 762, 286, 100, 30)

	w.controls[idGroupRepair] = createGroup(w.hwnd, "Repair Preparation", 16, 342, 720, 156)
	w.controls[idStaticInstaller] = createStatic(w.hwnd, "Installer path", 32, 370, 90, 22)
	w.controls[idInstaller] = createEdit(w.hwnd, "", 126, 368, 476, 24, esAutoHScroll)
	createButton(w, idBrowseInstaller, "Browse", 612, 366, 96, 28)
	w.controls[idStaticInvite] = createStatic(w.hwnd, "Invite", 32, 404, 90, 22)
	w.controls[idInvite] = createEdit(w.hwnd, "", 126, 402, 250, 24, esAutoHScroll|esPassword)
	w.controls[idStaticProfile] = createStatic(w.hwnd, "Profile", 400, 404, 55, 22)
	w.controls[idProfile] = createEdit(w.hwnd, emptyAs(w.activeProfile, "standard"), 458, 402, 150, 24, esAutoHScroll)
	w.controls[idStaticProfiles] = createStatic(w.hwnd, "Profiles: standard, conservative, diagnostic", 126, 434, 360, 22)
	createButton(w, idPreflight, "Preflight", 32, 460, 112, 28)
	createButton(w, idRepairDryRun, "Repair Dry-Run", 154, 460, 140, 28)

	w.controls[idGroupDanger] = createGroup(w.hwnd, "Danger Zone", 752, 342, 412, 156)
	w.controls[idStaticDanger1] = createStatic(w.hwnd, "Real repair modifies services, Defender exclusions, MSI state, and validated leftovers.", 770, 370, 370, 34)
	w.controls[idStaticDanger2] = createStatic(w.hwnd, "Requires Administrator, installer, invite, rollback snapshot, and exact YES.", 770, 408, 370, 22)
	createButton(w, idRepair, "RUN REAL REPAIR", 770, 448, 190, 38)

	w.controls[idGroupTimeline] = createGroup(w.hwnd, "Timeline", 16, 510, 720, 286)
	w.controls[idTimeline] = createEdit(w.hwnd, "", 32, 536, 688, 244, esMultiLine|esReadOnly|wsVScroll)

	w.controls[idGroupDetails] = createGroup(w.hwnd, "Details", 752, 510, 412, 286)
	w.controls[idReports] = createEdit(w.hwnd, "", 768, 536, 380, 170, esMultiLine|esReadOnly|wsVScroll|wsHScroll)
	createButton(w, idOpenSelectedReport, "Open Summary", 768, 716, 124, 28)
	createButton(w, idOpenOperations, "Open Operations", 900, 716, 128, 28)
	createButton(w, idCopyReportPath, "Copy Report", 1036, 716, 112, 28)
	createButton(w, idReportsList, "Reports List", 768, 752, 118, 28)
	createButton(w, idReportsCleanupDry, "Cleanup Dry-Run", 894, 752, 144, 28)
	createButton(w, idCopyBundlePath, "Copy Bundle", 1046, 752, 102, 28)
	enable(w.controls[idCancel], false)
	if w.admin {
		enable(w.controls[idRestartAdmin], false)
	}
}

func (w *mainWindow) windowProc(hwnd uintptr, msgID uint32, wparam uintptr, lparam uintptr) uintptr {
	switch msgID {
	case wmCommand:
		w.handleCommand(int(wparam & 0xffff))
		return 0
	case wmTimer:
		if w.controller.IsRunning() {
			w.updateTimelineFromProgress()
		}
		w.setRunningState(w.controller.IsRunning())
		return 0
	case wmSize:
		width := int(lparam & 0xffff)
		height := int((lparam >> 16) & 0xffff)
		w.layoutControls(width, height)
		return 0
	case wmGetMinMaxInfo:
		info := (*minMaxInfo)(unsafe.Pointer(lparam))
		info.MinTrackSize.X = minWindowWidth
		info.MinTrackSize.Y = minWindowHeight
		return 0
	case wmAppUI:
		w.runQueuedUI()
		return 0
	case wmClose, wmDestroy:
		w.controller.Cancel()
		procKillTimer.Call(hwnd, 1)
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msgID), wparam, lparam)
	return ret
}

func (w *mainWindow) handleCommand(id int) {
	defer w.watchUIAction(actionName(id), time.Now(), false)
	switch id {
	case idCheck:
		w.runAction(ActionCheck)
	case idVerify:
		w.runAction(ActionVerify)
	case idCollect:
		w.runAction(ActionCollectReport)
	case idReportsList:
		w.runAction(ActionReportsList)
	case idPreflight:
		w.runAction(ActionPreflight)
	case idRepairDryRun:
		w.runAction(ActionRepairDryRun)
	case idRepair:
		if !w.admin {
			w.showError("Real repair requires Administrator rights.")
			return
		}
		installer := strings.TrimSpace(getText(w.controls[idInstaller]))
		if installer == "" {
			w.showError("Select a valid Grabber MSI before real repair.")
			return
		}
		if !strings.EqualFold(filepath.Ext(installer), ".msi") {
			w.showError("Select a valid Grabber MSI before real repair.")
			return
		}
		if strings.TrimSpace(getText(w.controls[idInvite])) == "" {
			w.showError("Invite is required for real repair.")
			return
		}
		w.runAction(ActionRepair)
	case idReportsCleanupDry:
		w.runAction(ActionReportsCleanupDry)
	case idCancel:
		w.controller.Cancel()
		setText(w.controls[idInvite], "")
	case idOpenReports:
		target := w.latestReport
		if strings.TrimSpace(target) == "" {
			target = w.reportRoot
		}
		w.openPathAsync(target, "Reports folder is not available yet. Run Check first.")
	case idOpenSelectedReport:
		target := w.latestSummary
		if strings.TrimSpace(target) == "" {
			target = filepathFromReport(w.latestReport, "summary.txt")
		}
		w.openPathAsync(target, "summary.txt is not available for the latest run.")
	case idOpenOperations:
		target := filepathFromReport(w.latestReport, "operations.json")
		w.openPathAsync(target, "operations.json is not available for the latest run.")
	case idCopyReportPath:
		if strings.TrimSpace(w.latestReport) == "" {
			w.showError("Report path is not available yet. Run Check or Verify first.")
			return
		}
		if err := copyText(w.hwnd, w.latestReport); err != nil {
			w.showError(err.Error())
			return
		}
		messageBox(w.hwnd, "Report path copied.", "kigrepair GUI")
	case idCopyBundlePath:
		bundle := w.controller.Latest().SupportBundlePath
		if strings.TrimSpace(bundle) == "" {
			w.showError("Support bundle path is not available yet. Run Collect Bundle first.")
			return
		}
		if err := copyText(w.hwnd, bundle); err != nil {
			w.showError(err.Error())
			return
		}
		messageBox(w.hwnd, "Support bundle path copied.", "kigrepair GUI")
	case idBrowseInstaller:
		if path := openInstallerDialog(w.hwnd); path != "" {
			setText(w.controls[idInstaller], path)
			if warning := installerFilenameWarning(path); warning != "" {
				w.showError(warning)
			}
		}
	case idRestartAdmin:
		w.restartAsAdmin()
	}
}

func (w *mainWindow) runAction(action Action) {
	inputs := Inputs{
		InstallerPath: getText(w.controls[idInstaller]),
		InviteValue:   getText(w.controls[idInvite]),
		Profile:       getText(w.controls[idProfile]),
	}
	if (action == ActionPreflight || action == ActionRepairDryRun || action == ActionRepair) && strings.TrimSpace(inputs.InstallerPath) == "" {
		w.addLocalBlocking("Installer is required.")
		if action != ActionRepair {
			setText(w.controls[idInvite], "")
		}
		return
	}
	w.RunWorkflow(string(action), func(ctx context.Context, sink app.ProgressSink) (*app.WorkflowResponse, error) {
		_ = sink
		view, err := w.controller.Run(ctx, action, inputs)
		return responseFromView(view), err
	})
}

func (w *mainWindow) RunWorkflow(name string, run func(ctx context.Context, sink app.ProgressSink) (*app.WorkflowResponse, error)) {
	if w.controller.IsRunning() {
		w.showError("Another workflow is already running.")
		return
	}
	started := time.Now()
	progressBefore := w.controller.ProgressCount()
	uiDebug("workflow started: " + name)
	w.currentWorkflow = name
	w.setRunningState(true)
	w.updateDashboard(ResultView{Status: "Running", Workflow: name})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("workflow panic: %v", r)
				w.controller.mu.Lock()
				latest := w.controller.latest
				latest.Status = "Failed"
				latest.Workflow = name
				latest.Errors = append(latest.Errors, readableBlocker(err.Error(), err.Error()))
				w.controller.latest = latest
				w.controller.mu.Unlock()
				w.UI(func() {
					setText(w.controls[idInvite], "")
					w.updateDashboard(w.controller.Latest())
					w.currentWorkflow = ""
					w.setRunningState(false)
				})
			}
		}()
		_, err := run(context.Background(), w.controller.ProgressSink())
		if err != nil && !strings.Contains(err.Error(), "confirmation") {
			// The detailed, redacted error is also rendered from the controller result.
			w.controller.mu.Lock()
			latest := w.controller.latest
			latest.Errors = append(latest.Errors, readableBlocker(err.Error(), err.Error()))
			latest.BlockingReasons = append(latest.BlockingReasons, readableBlocker(err.Error(), err.Error()))
			w.controller.latest = latest
			w.controller.mu.Unlock()
		}
		progressAfter := w.controller.ProgressCount()
		latest := w.controller.Latest()
		uiDebug(fmt.Sprintf("workflow completed: %s duration_ms=%d progress_events=%d timeline_items=%d", name, time.Since(started).Milliseconds(), progressAfter-progressBefore, len(latest.Timeline)))
		w.UI(func() {
			setText(w.controls[idInvite], "")
			w.updateDashboard(w.controller.Latest())
			w.currentWorkflow = ""
			w.setRunningState(false)
		})
	}()
}

func (w *mainWindow) updateDashboard(view ResultView) {
	if view.ReportDir != "" {
		w.latestReport = view.ReportDir
	}
	if view.SummaryFile != "" {
		w.latestSummary = view.SummaryFile
	}
	info := version.Get()
	if view.Status == "" {
		view.Status = "Idle"
	}
	adminBadge := "Limited mode"
	if w.admin {
		adminBadge = "Admin"
	}
	header := []string{
		"kigrepair                                      " + info.Version + " / " + emptyAs(info.Commit, "dev"),
		"Kickidler Grabber Repair Utility",
		"Profile: " + emptyAs(getText(w.controls[idProfile]), emptyAs(w.activeProfile, "standard")) + "     Admin: " + adminBadge + "     Workflow: " + emptyAs(view.Workflow, "none") + "     Status: " + view.Status,
	}
	if !w.admin {
		header = append(header, "Limited mode: not running as Administrator. Service, Defender, and repair checks may be incomplete.")
	}
	setText(w.controls[idHeader], strings.Join(header, "\r\n"))
	setText(w.controls[idSystem], FormatSystemStatus(view))
	setText(w.controls[idTimeline], FormatTimeline(view.Timeline))
	reportLines := []string{
		"Latest report directory: " + emptyAs(view.ReportDir, "not available"),
		"summary.txt: " + emptyAs(view.SummaryFile, "not available"),
		"operations.json: " + emptyAs(view.OperationsFile, "not available"),
		"primary result: " + emptyAs(view.PrimaryResultFile, "not available"),
		"cleanup-plan.json: " + emptyAs(filepathFromReport(view.ReportDir, "cleanup-plan.json"), "not available"),
		"support bundle: " + emptyAs(view.SupportBundlePath, "not available"),
	}
	setText(w.controls[idReports], strings.Join(reportLines, "\r\n"))
	w.updateFileButtons(view)
}

func (w *mainWindow) updateTimelineFromProgress() {
	view := w.controller.Latest()
	view.Workflow = firstNonEmptyString(view.Workflow, w.currentWorkflow)
	view.Timeline = w.controller.ProgressTimeline()
	view.Timeline = SupportTimeline(view)
	setText(w.controls[idTimeline], FormatTimeline(view.Timeline))
}

func (w *mainWindow) setRunningState(running bool) {
	state := ComputeButtonState(running)
	for _, id := range []int{idCheck, idVerify, idCollect, idReportsList, idPreflight, idRepairDryRun, idRepair, idReportsCleanupDry, idBrowseInstaller, idRestartAdmin} {
		enable(w.controls[id], state.ActionsEnabled)
	}
	if w.admin {
		enable(w.controls[idRestartAdmin], false)
	}
	if w.relaunching {
		enable(w.controls[idRestartAdmin], false)
	}
	enable(w.controls[idCancel], state.CancelEnabled)
	w.updateFileButtons(w.controller.Latest())
}

func (w *mainWindow) confirm(ctx context.Context, req ConfirmationRequest) (bool, error) {
	type result struct {
		text string
	}
	ch := make(chan result, 1)
	w.UI(func() {
		ch <- result{text: promptText(w.hwnd, req.Title, req.Message)}
	})
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case res := <-ch:
		return AcceptExactYES(res.text), nil
	}
}

func (w *mainWindow) showError(message string) {
	messageBox(w.hwnd, safety.RedactString(message), "kigrepair GUI")
}

func (w *mainWindow) UI(fn func()) {
	if fn == nil {
		return
	}
	w.uiMu.Lock()
	w.uiQueue = append(w.uiQueue, fn)
	w.uiMu.Unlock()
	procPostMessage.Call(uintptr(w.hwnd), wmAppUI, 0, 0)
}

func (w *mainWindow) runQueuedUI() {
	w.uiMu.Lock()
	queue := append([]func(){}, w.uiQueue...)
	w.uiQueue = nil
	w.uiMu.Unlock()
	for _, fn := range queue {
		fn()
	}
}

func (w *mainWindow) detectAdminAsync() {
	admin := winapi.IsAdmin()
	w.UI(func() {
		w.admin = admin
		w.updateDashboard(w.controller.Latest())
		w.setRunningState(w.controller.IsRunning())
	})
}

func (w *mainWindow) addLocalBlocking(reason string) {
	view := w.controller.Latest()
	view.Status = "Not ready"
	view.BlockingReasons = dedupeStrings(append(view.BlockingReasons, reason))
	w.controller.mu.Lock()
	w.controller.latest = view
	w.controller.mu.Unlock()
	w.updateDashboard(view)
}

func (w *mainWindow) openPathAsync(path string, missingMessage string) {
	started := time.Now()
	go func() {
		if err := OpenPath(path); err != nil {
			w.UI(func() { w.showError(missingMessage) })
		}
		w.watchUIAction("shell_open", started, true)
	}()
}

func (w *mainWindow) restartAsAdmin() {
	if w.admin || w.elevatedChild || w.relaunching {
		return
	}
	if w.controller.IsRunning() {
		w.showError("Wait for the current workflow to finish or cancel it before restarting as Administrator.")
		return
	}
	if messageBoxYesNo(w.hwnd, "Restart kigrepair as Administrator?\r\n\r\nCurrent window will close.", "kigrepair GUI") != idYes {
		return
	}
	w.relaunching = true
	w.setRunningState(w.controller.IsRunning())
	setText(w.controls[idInvite], "")
	go func() {
		args := []string{"--elevated-child", "--no-auto-workflow"}
		if strings.TrimSpace(w.activeProfile) != "" {
			args = append(args, "--profile", w.activeProfile)
		}
		err := winapi.RelaunchElevated(args)
		if err != nil {
			w.UI(func() {
				w.relaunching = false
				w.setRunningState(w.controller.IsRunning())
				w.showError(err.Error())
			})
			return
		}
		w.UI(func() {
			procKillTimer.Call(uintptr(w.hwnd), 1)
			procDestroyWindow.Call(uintptr(w.hwnd))
		})
	}()
}

func responseFromView(view ResultView) *app.WorkflowResponse {
	return &app.WorkflowResponse{
		Meta: app.WorkflowResponseMeta{
			RunID:             view.RunID,
			Workflow:          view.Workflow,
			Status:            view.Status,
			ReportDir:         view.ReportDir,
			SummaryFile:       view.SummaryFile,
			OperationsFile:    view.OperationsFile,
			PrimaryResultFile: view.PrimaryResultFile,
		},
		Warnings: append([]string(nil), view.Warnings...),
		Errors:   append([]string(nil), view.Errors...),
	}
}

func createButton(w *mainWindow, id int, text string, x, y, width, height int) {
	w.controls[id] = createWindow("BUTTON", text, wsChild|wsVisible|wsTabStop|bsPushButton, x, y, width, height, w.hwnd, id)
}

func createGroup(parent windows.Handle, text string, x, y, width, height int) windows.Handle {
	return createWindow("BUTTON", text, wsChild|wsVisible|bsGroupBox, x, y, width, height, parent, 0)
}

func createStatic(parent windows.Handle, text string, x, y, width, height int) windows.Handle {
	return createWindow("STATIC", text, wsChild|wsVisible, x, y, width, height, parent, 0)
}

func createEdit(parent windows.Handle, text string, x, y, width, height int, style int) windows.Handle {
	return createWindow("EDIT", text, wsChild|wsVisible|wsBorder|wsTabStop|style, x, y, width, height, parent, 0)
}

func createWindow(class string, text string, style int, x, y, width, height int, parent windows.Handle, id int) windows.Handle {
	h, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(class))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		uintptr(parent),
		uintptr(id),
		0,
		0,
	)
	return windows.Handle(h)
}

func getText(hwnd windows.Handle) string {
	buf := make([]uint16, 4096)
	procGetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf)
}

func setText(hwnd windows.Handle, text string) {
	procSetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(utf16Ptr(safety.RedactString(text)))))
}

func enable(hwnd windows.Handle, enabled bool) {
	v := uintptr(0)
	if enabled {
		v = 1
	}
	procEnableWindow.Call(uintptr(hwnd), v)
}

func messageBox(parent windows.Handle, text, title string) {
	procMessageBox.Call(uintptr(parent), uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), 0)
}

func messageBoxYesNo(parent windows.Handle, text, title string) int {
	ret, _, _ := procMessageBox.Call(uintptr(parent), uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), mbYesNo)
	return int(ret)
}

func uiDebug(message string) {
	message = safety.RedactString(strings.TrimSpace(message))
	if message == "" {
		return
	}
	procOutputDebug.Call(uintptr(unsafe.Pointer(utf16Ptr("[kigrepair-gui] " + message))))
}

func utf16Ptr(value string) *uint16 {
	ptr, _ := windows.UTF16PtrFromString(value)
	return ptr
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type openFileName struct {
	StructSize    uint32
	Owner         windows.Handle
	Instance      windows.Handle
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustFilter uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxFileTitle  uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExt        *uint16
	CustData      uintptr
	FnHook        uintptr
	TemplateName  *uint16
	ReservedPtr   uintptr
	Reserved      uint32
	FlagsEx       uint32
}

func openInstallerDialog(owner windows.Handle) string {
	buffer := make([]uint16, windows.MAX_PATH)
	filter := multiStringPtr("MSI installers (*.msi)", "*.msi", "All files (*.*)", "*.*")
	title := utf16Ptr("Select Grabber MSI installer")
	ofn := openFileName{
		StructSize: uint32(unsafe.Sizeof(openFileName{})),
		Owner:      owner,
		Filter:     filter,
		File:       &buffer[0],
		MaxFile:    uint32(len(buffer)),
		Title:      title,
		Flags:      0x00001000 | 0x00000800,
	}
	ret, _, _ := procGetOpenFileName.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buffer)
}

func multiStringPtr(parts ...string) *uint16 {
	var values []uint16
	for _, part := range parts {
		values = append(values, syscall.StringToUTF16(part)...)
	}
	values = append(values, 0)
	return &values[0]
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

func copyText(owner windows.Handle, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("nothing to copy")
	}
	if ret, _, err := procOpenClipboard.Call(uintptr(owner)); ret == 0 {
		return fmt.Errorf("open clipboard failed: %w", err)
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	utf16 := syscall.StringToUTF16(text)
	bytes := uintptr(len(utf16) * 2)
	h, _, err := procGlobalAlloc.Call(0x0042, bytes)
	if h == 0 {
		return fmt.Errorf("clipboard allocation failed: %w", err)
	}
	ptr, _, err := procGlobalLock.Call(h)
	if ptr == 0 {
		return fmt.Errorf("clipboard lock failed: %w", err)
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16)), utf16)
	procGlobalUnlock.Call(h)
	if ret, _, err := procSetClipboardData.Call(13, h); ret == 0 {
		return fmt.Errorf("set clipboard data failed: %w", err)
	}
	return nil
}

func (w *mainWindow) layoutControls(width int, height int) {
	if w == nil || len(w.controls) == 0 {
		return
	}
	layout := computeMainLayout(width, height)
	move(w.controls[idHeader], layout.Header)
	move(w.controls[idGroupSystem], layout.SystemGroup)
	move(w.controls[idSystem], layout.System)
	move(w.controls[idGroupQuick], layout.QuickGroup)
	move(w.controls[idGroupRepair], layout.RepairGroup)
	move(w.controls[idGroupDanger], layout.DangerGroup)
	move(w.controls[idGroupTimeline], layout.TimelineGroup)
	move(w.controls[idTimeline], layout.Timeline)
	move(w.controls[idGroupDetails], layout.DetailsGroup)
	move(w.controls[idReports], layout.Details)
	for id, r := range layout.Buttons {
		move(w.controls[id], r)
	}
	for id, r := range layout.Labels {
		move(w.controls[id], r)
	}
}

func move(hwnd windows.Handle, r rect) {
	if hwnd == 0 {
		return
	}
	procSetWindowPos.Call(uintptr(hwnd), 0, uintptr(r.X), uintptr(r.Y), uintptr(r.W), uintptr(r.H), 0x0004)
}

func (w *mainWindow) updateFileButtons(view ResultView) {
	reportAvailable := strings.TrimSpace(view.ReportDir) != ""
	summaryAvailable := strings.TrimSpace(firstNonEmptyString(view.SummaryFile, filepathFromReport(view.ReportDir, "summary.txt"))) != ""
	operationsAvailable := strings.TrimSpace(filepathFromReport(view.ReportDir, "operations.json")) != ""
	bundleAvailable := strings.TrimSpace(view.SupportBundlePath) != ""
	enable(w.controls[idOpenReports], true)
	enable(w.controls[idOpenSelectedReport], reportAvailable && summaryAvailable)
	enable(w.controls[idOpenOperations], reportAvailable && operationsAvailable)
	enable(w.controls[idCopyReportPath], reportAvailable)
	enable(w.controls[idCopyBundlePath], bundleAvailable)
}

func (w *mainWindow) watchUIAction(name string, started time.Time, async bool) {
	elapsed := time.Since(started)
	level := "ui_action"
	if elapsed > 500*time.Millisecond {
		level = "ui_handler_slow"
	}
	uiDebug(fmt.Sprintf("%s name=%s async=%t duration_ms=%d", level, name, async, elapsed.Milliseconds()))
}

func actionName(id int) string {
	switch id {
	case idCheck:
		return "check"
	case idVerify:
		return "verify"
	case idCollect:
		return "collect_bundle"
	case idReportsList:
		return "reports_list"
	case idOpenReports:
		return "open_reports_folder"
	case idBrowseInstaller:
		return "browse_installer"
	case idPreflight:
		return "preflight"
	case idRepairDryRun:
		return "repair_dry_run"
	case idRepair:
		return "repair"
	case idCancel:
		return "cancel"
	case idOpenSelectedReport:
		return "open_summary"
	case idReportsCleanupDry:
		return "reports_cleanup_dry_run"
	case idRestartAdmin:
		return "restart_admin"
	case idOpenOperations:
		return "open_operations"
	case idCopyReportPath:
		return "copy_report_path"
	case idCopyBundlePath:
		return "copy_bundle_path"
	default:
		return fmt.Sprintf("command_%d", id)
	}
}

type minMaxInfo struct {
	Reserved     point
	MaxSize      point
	MaxPosition  point
	MinTrackSize point
	MaxTrackSize point
}
