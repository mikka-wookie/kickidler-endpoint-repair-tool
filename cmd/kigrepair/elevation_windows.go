package main

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"kigrepair/internal/checks"
)

const shellExecuteSuccessThreshold = 32

var shellExecuteW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

func ensureElevatedAtStart() error {
	if checks.IsAdmin() {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("admin elevation required, but executable path could not be resolved: %w", err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("admin elevation required, but working directory could not be resolved: %w", err)
	}

	if err := runElevated(exe, quoteCommandLineArgs(os.Args[1:]), workingDir); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func runElevated(exe string, args string, workingDir string) error {
	verbPtr, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	exePtr, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	argsPtr, err := windows.UTF16PtrFromString(args)
	if err != nil {
		return err
	}
	dirPtr, err := windows.UTF16PtrFromString(workingDir)
	if err != nil {
		return err
	}

	ret, _, callErr := shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		windows.SW_SHOWNORMAL,
	)
	if ret <= shellExecuteSuccessThreshold {
		if callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("admin elevation request failed: %w", callErr)
		}
		return fmt.Errorf("admin elevation request failed with ShellExecuteW code %d", ret)
	}
	return nil
}

func quoteCommandLineArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteCommandLineArg(arg))
	}
	return strings.Join(quoted, " ")
}

func quoteCommandLineArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			backslashes++
		case '"':
			b.WriteString(strings.Repeat(`\`, backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
		default:
			if backslashes > 0 {
				b.WriteString(strings.Repeat(`\`, backslashes))
				backslashes = 0
			}
			b.WriteRune(r)
		}
	}
	if backslashes > 0 {
		b.WriteString(strings.Repeat(`\`, backslashes*2))
	}
	b.WriteByte('"')
	return b.String()
}
