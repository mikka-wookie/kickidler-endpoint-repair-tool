//go:build windows

package ui

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

const guiMutexName = `Local\kigrepair-gui`

type singleInstanceGuard struct {
	handle windows.Handle
}

func acquireSingleInstance(allowMultiple bool, elevatedChild bool) (*singleInstanceGuard, error) {
	if allowMultiple {
		return &singleInstanceGuard{}, nil
	}
	var lastErr error
	deadline := time.Now().Add(5 * time.Second)
	for {
		handle, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(guiMutexName))
		if err != nil {
			return nil, fmt.Errorf("single-instance guard could not be created: %w", err)
		}
		if windows.GetLastError() != windows.ERROR_ALREADY_EXISTS {
			return &singleInstanceGuard{handle: handle}, nil
		}
		_ = windows.CloseHandle(handle)
		lastErr = errors.New("another kigrepair GUI instance is already running")
		if !elevatedChild || time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func (g *singleInstanceGuard) Release() {
	if g == nil || g.handle == 0 {
		return
	}
	_ = windows.ReleaseMutex(g.handle)
	_ = windows.CloseHandle(g.handle)
	g.handle = 0
}
