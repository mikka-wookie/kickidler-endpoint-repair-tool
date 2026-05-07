//go:build windows

package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
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
	wmCommand = 0x0111
	wmClose   = 0x0010
	wmDestroy = 0x0002
	wmTimer   = 0x0113
	wmAppDone = 0x8001

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
	wsVScroll     = 0x00200000
	wsTabStop     = 0x00010000
	wsGroup       = 0x00020000

	swShow = 5
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
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")
	procPostMessage      = user32.NewProc("PostMessageW")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGetOpenFileName  = comdlg32.NewProc("GetOpenFileNameW")
)

type mainWindow struct {
	hwnd          windows.Handle
	controller    *Controller
	controls      map[int]windows.Handle
	latestReport  string
	latestSummary string
	reportRoot    string
	configSource  string
	activeProfile string
	admin         bool
}

func runGUI() error {
	w := &mainWindow{controls: map[int]windows.Handle{}}
	w.controller = NewController(nil, w.confirm)
	w.controller.service = workflowservice.New(app.WorkflowOptions{ProgressSink: w.controller.ProgressSink()})
	effective, err := config.Load(config.LoadOptions{})
	if err == nil {
		w.configSource = effective.Metadata().Path
		w.activeProfile = effective.Config.Profile
		w.reportRoot = effective.Config.Reports.Root
	}
	if strings.TrimSpace(w.reportRoot) == "" {
		w.reportRoot = config.DefaultReportRoot
	}
	w.admin = winapi.IsAdmin()
	return w.run()
}

func (w *mainWindow) run() error {
	instance, _, _ := procGetModuleHandle.Call(0)
	className := utf16Ptr("KigrepairGuiWindow")
	wndproc := syscall.NewCallback(w.windowProc)
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   wndproc,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	if ret, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", err)
	}
	hwnd, _, err := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("kigrepair GUI MVP"))),
		wsOverlapped|wsCaption|wsSysMenu|wsThickFrame|wsMinimizeBox|wsMaximizeBox|wsVisible,
		100, 100, 980, 720,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", err)
	}
	w.hwnd = windows.Handle(hwnd)
	w.createControls()
	w.updateDashboard(ResultView{Status: "idle"})
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	procSetTimer.Call(hwnd, 1, 750, 0)

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
	w.controls[idHeader] = createEdit(w.hwnd, "", 16, 12, 948, 70, esMultiLine|esReadOnly)

	createGroup(w.hwnd, "System Status", 16, 92, 948, 128)
	w.controls[idSystem] = createEdit(w.hwnd, "", 32, 118, 916, 86, esMultiLine|esReadOnly|wsVScroll)

	createGroup(w.hwnd, "Quick Actions", 16, 230, 948, 70)
	createButton(w, idCheck, "Check", 32, 256, 95, 30)
	createButton(w, idVerify, "Verify", 134, 256, 95, 30)
	createButton(w, idCollect, "Collect Bundle", 236, 256, 125, 30)
	createButton(w, idOpenReports, "Open Reports Folder", 368, 256, 160, 30)
	createButton(w, idRestartAdmin, "Restart as Administrator", 536, 256, 180, 30)
	createButton(w, idCancel, "Cancel", 724, 256, 90, 30)

	createGroup(w.hwnd, "Repair Preparation", 16, 310, 620, 140)
	createStatic(w.hwnd, "Installer:", 32, 338, 70, 22)
	w.controls[idInstaller] = createEdit(w.hwnd, "", 104, 336, 410, 24, esAutoHScroll)
	createButton(w, idBrowseInstaller, "Browse", 522, 334, 90, 28)
	createStatic(w.hwnd, "Invite:", 32, 370, 70, 22)
	w.controls[idInvite] = createEdit(w.hwnd, "", 104, 368, 250, 24, esAutoHScroll|esPassword)
	createStatic(w.hwnd, "Profile:", 370, 370, 70, 22)
	w.controls[idProfile] = createEdit(w.hwnd, emptyAs(w.activeProfile, "standard"), 438, 368, 120, 24, esAutoHScroll)
	createStatic(w.hwnd, "standard / conservative / diagnostic", 104, 398, 300, 22)
	createButton(w, idPreflight, "Preflight", 32, 414, 100, 28)
	createButton(w, idRepairDryRun, "Repair Dry-Run", 140, 414, 130, 28)

	createGroup(w.hwnd, "Danger Zone", 650, 310, 314, 140)
	createStatic(w.hwnd, "Real repair can stop services, remove validated leftovers,", 666, 338, 280, 18)
	createStatic(w.hwnd, "add Defender exclusions, and reinstall Grabber.", 666, 356, 280, 18)
	createButton(w, idRepair, "RUN REAL REPAIR", 666, 392, 170, 38)

	createGroup(w.hwnd, "Timeline", 16, 460, 948, 146)
	w.controls[idTimeline] = createEdit(w.hwnd, "", 32, 486, 916, 104, esMultiLine|esReadOnly|wsVScroll)

	createGroup(w.hwnd, "Details", 16, 616, 948, 82)
	w.controls[idReports] = createEdit(w.hwnd, "", 32, 642, 540, 40, esMultiLine|esReadOnly|wsVScroll)
	createButton(w, idOpenSelectedReport, "Open Summary", 588, 642, 120, 28)
	createButton(w, idOpenOperations, "Open Operations", 714, 642, 126, 28)
	createButton(w, idCopyReportPath, "Copy Report Path", 846, 642, 102, 28)
	createButton(w, idReportsList, "Reports List", 588, 672, 110, 24)
	createButton(w, idReportsCleanupDry, "Cleanup Dry-Run", 704, 672, 140, 24)
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
		w.setRunningState(w.controller.IsRunning())
		return 0
	case wmAppDone:
		w.updateDashboard(w.controller.Latest())
		w.setRunningState(false)
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
			w.showError("Administrator rights are required for real repair. Use Restart as Administrator, then run Real Repair again.")
			return
		}
		if strings.TrimSpace(getText(w.controls[idInstaller])) == "" {
			w.showError("Installer file was not found. Select a supported MSI before real repair.")
			return
		}
		if strings.TrimSpace(getText(w.controls[idInvite])) == "" {
			w.showError("Invite is required for this workflow.")
			return
		}
		w.runAction(ActionRepair)
	case idReportsCleanupDry:
		w.runAction(ActionReportsCleanupDry)
	case idCancel:
		w.controller.Cancel()
		setText(w.controls[idInvite], "")
	case idOpenReports:
		target := w.reportRoot
		if strings.TrimSpace(target) == "" {
			target = w.latestReport
		}
		if err := OpenPath(target); err != nil {
			w.showError(err.Error())
		}
	case idOpenSelectedReport:
		target := w.latestSummary
		if strings.TrimSpace(target) == "" {
			target = w.latestReport
		}
		if err := OpenPath(target); err != nil {
			w.showError(err.Error())
		}
	case idOpenOperations:
		target := filepathFromReport(w.latestReport, "operations.json")
		if err := OpenPath(target); err != nil {
			w.showError("Operations file is not available yet. Run Check or Verify first.")
		}
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
	case idBrowseInstaller:
		if path := openInstallerDialog(w.hwnd); path != "" {
			setText(w.controls[idInstaller], path)
			if warning := installerFilenameWarning(path); warning != "" {
				w.showError(warning)
			}
		}
	case idRestartAdmin:
		if err := winapi.RelaunchElevated(nil); err != nil {
			w.showError(err.Error())
		}
	}
}

func (w *mainWindow) runAction(action Action) {
	inputs := Inputs{
		InstallerPath: getText(w.controls[idInstaller]),
		InviteValue:   getText(w.controls[idInvite]),
		Profile:       getText(w.controls[idProfile]),
	}
	w.setRunningState(true)
	setText(w.controls[idHeader], "kigrepair\r\nKickidler Grabber Repair Utility\r\nStatus: Running "+string(action)+"...")
	go func() {
		_, err := w.controller.Run(context.Background(), action, inputs)
		setText(w.controls[idInvite], "")
		if err != nil && !strings.Contains(err.Error(), "confirmation") {
			// The detailed, redacted error is also rendered from the controller result.
			w.controller.mu.Lock()
			latest := w.controller.latest
			latest.Errors = append(latest.Errors, readableBlocker(err.Error(), err.Error()))
			latest.BlockingReasons = append(latest.BlockingReasons, readableBlocker(err.Error(), err.Error()))
			w.controller.latest = latest
			w.controller.mu.Unlock()
		}
		procPostMessage.Call(uintptr(w.hwnd), wmAppDone, 0, 0)
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
		"Profile: " + emptyAs(getText(w.controls[idProfile]), emptyAs(w.activeProfile, "standard")) + "     Admin: " + adminBadge + "     Status: " + view.Status,
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
}

func (w *mainWindow) setRunningState(running bool) {
	state := ComputeButtonState(running)
	for _, id := range []int{idCheck, idVerify, idCollect, idReportsList, idPreflight, idRepairDryRun, idRepair, idReportsCleanupDry, idBrowseInstaller, idRestartAdmin} {
		enable(w.controls[id], state.ActionsEnabled)
	}
	if w.admin {
		enable(w.controls[idRestartAdmin], false)
	}
	enable(w.controls[idCancel], state.CancelEnabled)
}

func (w *mainWindow) confirm(ctx context.Context, req ConfirmationRequest) (bool, error) {
	text := promptText(w.hwnd, req.Title, req.Message)
	return AcceptExactYES(text), nil
}

func (w *mainWindow) showError(message string) {
	messageBox(w.hwnd, safety.RedactString(message), "kigrepair GUI")
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
