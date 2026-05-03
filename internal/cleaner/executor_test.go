package cleaner

import (
	"errors"
	"os"
	"testing"
	"time"

	"kigrepair/internal/app"
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

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "TeleLinkSoft" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return os.ModeDir }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return true }
func (fakeFileInfo) Sys() any           { return nil }
