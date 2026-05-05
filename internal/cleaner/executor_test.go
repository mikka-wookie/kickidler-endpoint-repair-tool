package cleaner

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/safety"
)

func TestDeletePathExecutorRefusesUnknownPath(t *testing.T) {
	executor := testExecutor()

	result := executor.ExecuteAction(CleanupAction{
		Type:   CleanupActionDeletePath,
		Target: `C:\Unknown`,
		Safe:   true,
	})

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed", result.Status)
	}
	if result.Error == "" {
		t.Fatalf("expected safety error")
	}
}

func TestDeletePathExecutorRefusesDriveRoot(t *testing.T) {
	executor := testExecutor()

	result := executor.ExecuteAction(CleanupAction{
		Type:   CleanupActionDeletePath,
		Target: `C:\`,
		Safe:   true,
	})

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed", result.Status)
	}
}

func TestDeletePathExecutorAllowsValidatedPath(t *testing.T) {
	removed := ""
	executor := testExecutor()
	executor.ValidatePath = func(path string) error {
		if path != `C:\Program Files\TeleLinkSoft` {
			return safety.ErrUnknownPath
		}
		return nil
	}
	executor.RemoveAll = func(path string) error {
		removed = path
		return nil
	}
	statCalls := 0
	executor.Stat = func(string) (os.FileInfo, error) {
		statCalls++
		if statCalls == 1 {
			return fakeFileInfo{}, nil
		}
		return nil, os.ErrNotExist
	}

	result := executor.ExecuteAction(CleanupAction{
		Type:   CleanupActionDeletePath,
		Target: `C:\Program Files\TeleLinkSoft`,
		Safe:   true,
	})

	if result.Status != app.OperationStatusSuccess {
		t.Fatalf("status = %s, want success: %#v", result.Status, result)
	}
	if removed != `C:\Program Files\TeleLinkSoft` {
		t.Fatalf("removed path = %q", removed)
	}
}

func TestDeleteRegistryKeyExecutorRefusesUnknownKey(t *testing.T) {
	executor := testExecutor()

	result := executor.ExecuteAction(CleanupAction{
		Type:   CleanupActionDeleteRegistryKey,
		Target: `HKLM\SOFTWARE\Unknown`,
		Safe:   true,
	})

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed", result.Status)
	}
}

func TestKillProcessRefusesUntrustedPlan(t *testing.T) {
	executor := testExecutor()

	result := executor.ExecuteAction(CleanupAction{
		Type:           CleanupActionKillProcess,
		Target:         `grabber.exe PID 10 C:\Temp\grabber.exe`,
		Safe:           true,
		PID:            10,
		ProcessName:    "grabber.exe",
		ExecutablePath: `C:\Temp\grabber.exe`,
		TrustLevel:     detector.ProcessTrustPathMismatch,
	})

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed", result.Status)
	}
}

func TestKillProcessAlreadyExitedIsSkipped(t *testing.T) {
	executor := testProcessExecutor(func(name string, args ...string) CommandResult {
		if name == "powershell.exe" {
			return CommandResult{Output: "null"}
		}
		return CommandResult{}
	})

	result := executor.ExecuteAction(trustedProcessAction(10, `C:\Program Files\TeleLinkSoft\bin\grabber.exe`))

	if result.Status != app.OperationStatusSkipped {
		t.Fatalf("status = %s, want skipped: %#v", result.Status, result)
	}
}

func TestKillProcessPIDReusedDifferentPathFails(t *testing.T) {
	executor := testProcessExecutor(func(name string, args ...string) CommandResult {
		if name == "powershell.exe" {
			return CommandResult{Output: processJSON(10, "grabber.exe", `C:\Temp\grabber.exe`)}
		}
		return CommandResult{}
	})

	result := executor.ExecuteAction(trustedProcessAction(10, `C:\Program Files\TeleLinkSoft\bin\grabber.exe`))

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed: %#v", result.Status, result)
	}
	if !strings.Contains(result.Error, "path changed") {
		t.Fatalf("error = %q, want path changed", result.Error)
	}
}

func TestKillProcessAccessDeniedReturnsAdminMessage(t *testing.T) {
	queryCount := 0
	executor := testProcessExecutor(func(name string, args ...string) CommandResult {
		switch name {
		case "powershell.exe":
			queryCount++
			return CommandResult{Output: processJSON(10, "grabber.exe", `C:\Program Files\TeleLinkSoft\bin\grabber.exe`)}
		case "taskkill.exe":
			return CommandResult{ExitCode: 5, Output: "ERROR: Access is denied."}
		default:
			return CommandResult{}
		}
	})

	result := executor.ExecuteAction(trustedProcessAction(10, `C:\Program Files\TeleLinkSoft\bin\grabber.exe`))

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed: %#v", result.Status, result)
	}
	if !strings.Contains(result.Message, "Administrator") {
		t.Fatalf("message = %q, want Administrator hint", result.Message)
	}
	if queryCount != 1 {
		t.Fatalf("query count = %d, want 1", queryCount)
	}
}

func TestKillProcessTimeoutFails(t *testing.T) {
	executor := testProcessExecutor(func(name string, args ...string) CommandResult {
		switch name {
		case "powershell.exe":
			return CommandResult{Output: processJSON(10, "grabber.exe", `C:\Program Files\TeleLinkSoft\bin\grabber.exe`)}
		case "taskkill.exe":
			return CommandResult{ExitCode: 0}
		default:
			return CommandResult{}
		}
	})
	executor.ProcessTimeout = time.Millisecond
	executor.ProcessPoll = time.Millisecond

	result := executor.ExecuteAction(trustedProcessAction(10, `C:\Program Files\TeleLinkSoft\bin\grabber.exe`))

	if result.Status != app.OperationStatusFailed {
		t.Fatalf("status = %s, want failed: %#v", result.Status, result)
	}
	if !strings.Contains(result.Message, "Timed out") {
		t.Fatalf("message = %q, want timeout", result.Message)
	}
}

func TestKillProcessRevalidatesAndTerminatesByPIDOnly(t *testing.T) {
	taskkillArgs := []string{}
	queryCount := 0
	executor := testProcessExecutor(func(name string, args ...string) CommandResult {
		switch name {
		case "powershell.exe":
			queryCount++
			if queryCount == 1 {
				return CommandResult{Output: processJSON(10, "grabber.exe", `C:\Program Files\TeleLinkSoft\bin\grabber.exe`)}
			}
			return CommandResult{Output: "null"}
		case "taskkill.exe":
			taskkillArgs = append([]string{}, args...)
			return CommandResult{ExitCode: 0}
		default:
			return CommandResult{}
		}
	})

	result := executor.ExecuteAction(trustedProcessAction(10, `C:\Program Files\TeleLinkSoft\bin\grabber.exe`))

	if result.Status != app.OperationStatusSuccess {
		t.Fatalf("status = %s, want success: %#v", result.Status, result)
	}
	if strings.Join(taskkillArgs, " ") != "/PID 10 /F" {
		t.Fatalf("taskkill args = %#v", taskkillArgs)
	}
}

func testExecutor() Executor {
	return Executor{
		RunCommand: func(string, ...string) CommandResult {
			return CommandResult{}
		},
		RemoveAll: func(string) error {
			return nil
		},
		Stat: func(string) (os.FileInfo, error) {
			return nil, errors.New("stat should not be called")
		},
		ValidatePath: safety.ValidateCleanupPath,
	}
}

func testProcessExecutor(run CommandRunner) Executor {
	executor := testExecutor()
	executor.RunCommand = run
	executor.ProcessTimeout = 50 * time.Millisecond
	executor.ProcessPoll = time.Millisecond
	return executor
}

func trustedProcessAction(pid int, path string) CleanupAction {
	return CleanupAction{
		Type:           CleanupActionKillProcess,
		Target:         "grabber.exe PID 10 " + path,
		Safe:           true,
		PID:            pid,
		ProcessName:    "grabber.exe",
		ExecutablePath: path,
		TrustLevel:     detector.ProcessTrustNameAndPathMatch,
		MatchReason:    "known Grabber process name under allowlisted install root",
	}
}

func processJSON(pid int, name string, path string) string {
	return `{"ProcessId":` + strconv.Itoa(pid) + `,"Name":"` + name + `","ExecutablePath":"` + strings.ReplaceAll(path, `\`, `\\`) + `","CommandLine":""}`
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "TeleLinkSoft" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return os.ModeDir }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return true }
func (fakeFileInfo) Sys() any           { return nil }
