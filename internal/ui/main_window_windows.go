//go:build windows

package ui

import (
	"context"
	"fmt"
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
	idSummary
	idReports
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	comdlg32            = windows.NewLazySystemDLL("comdlg32.dll")
	procRegisterClassEx = user32.NewProc("RegisterClassExW")
	procCreateWindowEx  = user32.NewProc("CreateWindowExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procShowWindow      = user32.NewProc("ShowWindow")
	procUpdateWindow    = user32.NewProc("UpdateWindow")
	procGetMessage      = user32.NewProc("GetMessageW")
	procTranslateMsg    = user32.NewProc("TranslateMessage")
	procDispatchMsg     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage = user32.NewProc("PostQuitMessage")
	procSetWindowText   = user32.NewProc("SetWindowTextW")
	procGetWindowText   = user32.NewProc("GetWindowTextW")
	procEnableWindow    = user32.NewProc("EnableWindow")
	procMessageBox      = user32.NewProc("MessageBoxW")
	procSendMessage     = user32.NewProc("SendMessageW")
	procSetTimer        = user32.NewProc("SetTimer")
	procKillTimer       = user32.NewProc("KillTimer")
	procPostMessage     = user32.NewProc("PostMessageW")
	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
	procGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")
)

type mainWindow struct {
	hwnd          windows.Handle
	controller    *Controller
	controls      map[int]windows.Handle
	latestReport  string
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
	createStatic(w.hwnd, "Status / Dashboard", 16, 12, 220, 22)
	createButton(w, idCheck, "Check", 16, 40, 95, 28)
	createButton(w, idVerify, "Verify", 118, 40, 95, 28)
	createButton(w, idCollect, "Collect Report", 220, 40, 120, 28)
	createButton(w, idOpenReports, "Open Reports Folder", 348, 40, 150, 28)
	createButton(w, idRestartAdmin, "Restart as Administrator", 506, 40, 170, 28)

	createStatic(w.hwnd, "Repair Preparation", 16, 86, 220, 22)
	createStatic(w.hwnd, "Installer:", 16, 116, 70, 22)
	w.controls[idInstaller] = createEdit(w.hwnd, "", 88, 114, 560, 24, esAutoHScroll)
	createButton(w, idBrowseInstaller, "Browse installer", 656, 112, 125, 28)
	createStatic(w.hwnd, "Invite:", 16, 148, 70, 22)
	w.controls[idInvite] = createEdit(w.hwnd, "", 88, 146, 300, 24, esAutoHScroll|esPassword)
	createStatic(w.hwnd, "Profile:", 408, 148, 70, 22)
	w.controls[idProfile] = createEdit(w.hwnd, "standard", 480, 146, 110, 24, esAutoHScroll)
	createStatic(w.hwnd, "standard / conservative / diagnostic", 600, 148, 250, 22)
	createButton(w, idPreflight, "Preflight", 16, 184, 100, 28)
	createButton(w, idRepairDryRun, "Repair Dry-Run", 124, 184, 125, 28)
	createButton(w, idRepair, "Real Repair", 276, 184, 120, 32)
	createButton(w, idCancel, "Cancel", 404, 184, 90, 32)

	createStatic(w.hwnd, "Results / Timeline", 16, 232, 220, 22)
	w.controls[idSummary] = createEdit(w.hwnd, "", 16, 260, 930, 116, esMultiLine|esReadOnly|wsVScroll)
	w.controls[idTimeline] = createEdit(w.hwnd, "", 16, 386, 930, 190, esMultiLine|esReadOnly|wsVScroll)

	createStatic(w.hwnd, "Reports", 16, 590, 220, 22)
	w.controls[idReports] = createEdit(w.hwnd, "", 16, 616, 610, 44, esMultiLine|esReadOnly|wsVScroll)
	createButton(w, idOpenSelectedReport, "Open report folder", 642, 616, 140, 28)
	createButton(w, idReportsList, "Reports List", 790, 616, 100, 28)
	createButton(w, idReportsCleanupDry, "Cleanup Dry-Run", 642, 650, 140, 28)
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
			w.showError("Administrator rights are required for real repair. Restart as Administrator, then run Real Repair again.")
			return
		}
		w.runAction(ActionRepair)
	case idReportsCleanupDry:
		w.runAction(ActionReportsCleanupDry)
	case idCancel:
		w.controller.Cancel()
	case idOpenReports, idOpenSelectedReport:
		if err := OpenPath(w.latestReport); err != nil {
			w.showError(err.Error())
		}
	case idBrowseInstaller:
		if path := openInstallerDialog(w.hwnd); path != "" {
			setText(w.controls[idInstaller], path)
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
	setText(w.controls[idSummary], "Status: running\r\nWorkflow: "+string(action))
	go func() {
		_, err := w.controller.Run(context.Background(), action, inputs)
		setText(w.controls[idInvite], "")
		if err != nil && !strings.Contains(err.Error(), "confirmation") {
			// The detailed, redacted error is also rendered from the controller result.
			w.controller.mu.Lock()
			latest := w.controller.latest
			latest.Errors = append(latest.Errors, safety.RedactString(err.Error()))
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
	info := version.Get()
	admin := "no"
	if w.admin {
		admin = "yes"
	}
	if view.Status == "" {
		view.Status = "idle"
	}
	prefix := []string{
		"App version: " + info.Version,
		"Active profile: " + emptyAs(w.activeProfile, emptyAs(getText(w.controls[idProfile]), "standard")),
		"Config source: " + emptyAs(w.configSource, "defaults"),
		"Admin: " + admin,
	}
	setText(w.controls[idSummary], strings.Join(prefix, "\r\n")+"\r\n"+FormatResultSummary(view))
	setText(w.controls[idTimeline], FormatTimeline(view.Timeline))
	reportLines := []string{
		"Latest report directory: " + emptyAs(view.ReportDir, "not available"),
		"summary.txt: " + emptyAs(view.SummaryFile, "not available"),
		"operations.json: " + emptyAs(view.OperationsFile, "not available"),
		"primary result: " + emptyAs(view.PrimaryResultFile, "not available"),
		"support bundle: " + emptyAs(view.SupportBundlePath, "not available"),
	}
	setText(w.controls[idReports], strings.Join(reportLines, "\r\n"))
}

func (w *mainWindow) setRunningState(running bool) {
	for _, id := range []int{idCheck, idVerify, idCollect, idReportsList, idPreflight, idRepairDryRun, idRepair, idReportsCleanupDry, idBrowseInstaller, idRestartAdmin} {
		enable(w.controls[id], !running)
	}
	enable(w.controls[idCancel], running)
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
