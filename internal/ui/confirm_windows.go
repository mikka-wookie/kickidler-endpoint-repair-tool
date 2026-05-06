//go:build windows

package ui

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	confirmOK     = 7101
	confirmCancel = 7102
	confirmEdit   = 7103
)

type confirmWindow struct {
	hwnd   windows.Handle
	edit   windows.Handle
	done   bool
	result string
}

func promptText(owner windows.Handle, title, message string) string {
	cw := &confirmWindow{}
	instance, _, _ := procGetModuleHandle.Call(0)
	className := utf16Ptr("KigrepairConfirmWindow")
	wndproc := syscall.NewCallback(cw.windowProc)
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   wndproc,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		wsCaption|wsSysMenu|wsVisible,
		180, 180, 560, 260,
		uintptr(owner), 0, instance, 0,
	)
	cw.hwnd = windows.Handle(hwnd)
	createStatic(cw.hwnd, message, 16, 14, 510, 120)
	createStatic(cw.hwnd, "Type YES:", 16, 148, 80, 22)
	cw.edit = createEdit(cw.hwnd, "", 100, 146, 120, 24, esAutoHScroll)
	createWindow("BUTTON", "OK", wsChild|wsVisible|wsTabStop|bsPushButton, 300, 184, 90, 28, cw.hwnd, confirmOK)
	createWindow("BUTTON", "Cancel", wsChild|wsVisible|wsTabStop|bsPushButton, 400, 184, 90, 28, cw.hwnd, confirmCancel)

	var msg msg
	for !cw.done {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return cw.result
}

func (cw *confirmWindow) windowProc(hwnd uintptr, msgID uint32, wparam uintptr, lparam uintptr) uintptr {
	switch msgID {
	case wmCommand:
		switch int(wparam & 0xffff) {
		case confirmOK:
			cw.result = getText(cw.edit)
			cw.done = true
			procShowWindow.Call(hwnd, 0)
			return 0
		case confirmCancel:
			cw.result = ""
			cw.done = true
			procShowWindow.Call(hwnd, 0)
			return 0
		}
	case wmClose, wmDestroy:
		cw.result = ""
		cw.done = true
		procShowWindow.Call(hwnd, 0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msgID), wparam, lparam)
	return ret
}
