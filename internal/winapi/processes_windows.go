//go:build windows

package winapi

import (
	"context"
	"errors"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func listProcesses(ctx context.Context) ([]RawProcessInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}
	processes := make([]RawProcessInfo, 0, 256)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pid := int(entry.ProcessID)
		name := windows.UTF16ToString(entry.ExeFile[:])
		processes = append(processes, RawProcessInfo{
			ProcessID:      pid,
			Name:           name,
			ExecutablePath: processImagePath(pid),
		})
		err := windows.Process32Next(snapshot, &entry)
		if errors.Is(err, syscall.ERROR_NO_MORE_FILES) {
			break
		}
		if err != nil {
			return processes, err
		}
	}
	return processes, nil
}

func processImagePath(pid int) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
		buffer = make([]uint16, 32768)
		size = uint32(len(buffer))
		if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
			return ""
		}
	}
	return filepath.Clean(windows.UTF16ToString(buffer[:size]))
}

func terminateProcessByPID(ctx context.Context, pid int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return nil
		}
		return err
	}
	defer windows.CloseHandle(handle)
	if err := windows.TerminateProcess(handle, 1); err != nil {
		return err
	}
	return nil
}
