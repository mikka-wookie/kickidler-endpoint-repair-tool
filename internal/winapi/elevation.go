//go:build windows

package winapi

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const shellExecuteSuccessThreshold = 32

var shellExecuteW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

func IsAdmin() bool {
	var sid *windows.SID
	if err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	); err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	return err == nil && member
}

func RelaunchElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable path could not be resolved: %w", err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("working directory could not be resolved: %w", err)
	}

	childArgs := appendElevatedChildArg(args)
	return shellExecuteRunas(exe, QuoteWindowsArgs(childArgs), workingDir)
}

func shellExecuteRunas(exe string, args string, workingDir string) error {
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
			return fmt.Errorf("ShellExecuteW runas failed: %w", callErr)
		}
		return fmt.Errorf("ShellExecuteW runas failed with code %d", ret)
	}
	return nil
}

func QuoteWindowsArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, QuoteWindowsArg(arg))
	}
	return strings.Join(quoted, " ")
}

func QuoteWindowsArg(arg string) string {
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

func appendElevatedChildArg(args []string) []string {
	for _, arg := range args {
		if arg == "--elevated-child" || strings.HasPrefix(arg, "--elevated-child=") {
			return append([]string(nil), args...)
		}
	}
	childArgs := append([]string(nil), args...)
	childArgs = append(childArgs, "--elevated-child")
	return childArgs
}
