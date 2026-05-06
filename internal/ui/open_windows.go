//go:build windows

package ui

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openPath(path string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	target, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	ret, _, callErr := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW").Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(target)),
		0,
		0,
		windows.SW_SHOWNORMAL,
	)
	if ret <= 32 {
		if callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("open path failed: %w", callErr)
		}
		return fmt.Errorf("open path failed with code %d", ret)
	}
	return nil
}
