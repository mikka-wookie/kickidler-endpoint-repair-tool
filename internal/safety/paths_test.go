package safety

import (
	"errors"
	"testing"
)

func setCleanupPathEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)
	t.Setenv("ProgramData", `C:\ProgramData`)
	t.Setenv("SystemRoot", `C:\Windows`)
}

func TestValidateCleanupPathAllowsExactConfiguredPaths(t *testing.T) {
	setCleanupPathEnv(t)

	tests := []string{
		`C:\Program Files\TeleLinkSoft`,
		`C:\Program Files\TeleLinkSoftHelper`,
		`C:\Program Files (x86)\TeleLinkSoft`,
		`C:\Program Files (x86)\TeleLinkSoftHelper`,
		`C:\ProgramData\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60`,
		`C:\Windows\System32\wmi`,
	}

	for _, path := range tests {
		if err := ValidateCleanupPath(path); err != nil {
			t.Fatalf("ValidateCleanupPath(%q) returned error: %v", path, err)
		}
	}
}

func TestValidateCleanupPathRejectsUnsafePaths(t *testing.T) {
	setCleanupPathEnv(t)

	tests := []struct {
		name string
		path string
		want error
	}{
		{"drive root", `C:\`, ErrDriveRoot},
		{"windows root", `C:\Windows`, ErrUnknownPath},
		{"system32 root", `C:\Windows\System32`, ErrUnknownPath},
		{"unknown path", `C:\Program Files\Unknown`, ErrUnknownPath},
		{"relative path", `TeleLinkSoft`, ErrRelativePath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCleanupPath(tt.path)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ValidateCleanupPath(%q) = %v, want %v", tt.path, err, tt.want)
			}
		})
	}
}

func TestValidateCleanupPathRejectsUnresolvedEnvironmentPath(t *testing.T) {
	setCleanupPathEnv(t)
	t.Setenv("SystemRoot", "")

	err := ValidateCleanupPath(`%SystemRoot%\System32\wmi`)
	if !errors.Is(err, ErrUnresolvedEnv) {
		t.Fatalf("ValidateCleanupPath returned %v, want %v", err, ErrUnresolvedEnv)
	}
}
