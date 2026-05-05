package preflight_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/preflight"
	"kigrepair/internal/reports"
	"kigrepair/internal/testfixtures"
)

func TestPreflightScenarioReadiness(t *testing.T) {
	tests := []struct {
		name      string
		hasInvite bool
		want      string
	}{
		{name: "missing_invite_blocks_repair_readiness", hasInvite: false, want: preflight.StatusNotReady},
		{name: "missing_admin_blocks_repair_readiness", hasInvite: true, want: preflight.StatusNotReady},
		{name: "defender_query_failed", hasInvite: true, want: preflight.StatusReadyWithWarnings},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := testfixtures.ScenarioByName(tt.name)
			ctx := preflightContext(t)
			result, err := preflight.Run(ctx, preflight.Options{
				InstallerPath: scenario.Installer.Path,
				HasInvite:     tt.hasInvite,
				OutputDir:     ctx.OutputDir,
				Quiet:         true,
			}, depsForScenario(scenario))
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if result.Status != tt.want {
				t.Fatalf("status = %s, want %s; failed=%v warnings=%v", result.Status, tt.want, result.FailedRequiredChecks, result.Warnings)
			}
			if tt.name == "defender_query_failed" && !contains(result.Warnings, "defender_read") {
				t.Fatalf("warnings = %#v, want defender_read", result.Warnings)
			}
		})
	}
}

func depsForScenario(scenario testfixtures.Scenario) preflight.Dependencies {
	return preflight.Dependencies{
		Now:          func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) },
		IsAdmin:      func() bool { return scenario.IsAdmin },
		OS:           func() string { return "windows" },
		Architecture: func() string { return "amd64" },
		ExePath:      func() (string, error) { return `C:\Support\kigrepair.exe`, nil },
		WorkingDir:   func() (string, error) { return `C:\Support`, nil },
		LookPath: func(name string) (string, error) {
			switch name {
			case "powershell.exe", "msiexec.exe":
				return `C:\Windows\System32\` + name, nil
			default:
				return "", errors.New("not found")
			}
		},
		ResolveInstaller: func(string) preflight.InstallerResolution {
			return preflight.InstallerResolution{Path: scenario.Installer.Path, Provided: true}
		},
		Detect: func() (detector.DetectionReport, error) {
			return scenario.DetectionReport(), nil
		},
		ReadFile: func(path string) error {
			if scenario.Installer.Exists {
				return nil
			}
			return os.ErrNotExist
		},
		HashFile:   func(string) (string, error) { return "fixture-sha256", nil },
		WriteProbe: func(string) error { return nil },
	}
}

func preflightContext(t *testing.T) *app.AppContext {
	t.Helper()
	dir := t.TempDir()
	reporter, err := reports.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := app.NewContext()
	ctx.OutputDir = dir
	ctx.Reporter = reporter
	ctx.Logger = logging.Discard()
	ctx.Quiet = true
	return ctx
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestPreflightDoesNotWriteRawInvite(t *testing.T) {
	scenario := testfixtures.ScenarioByName("missing_invite_blocks_repair_readiness")
	ctx := preflightContext(t)
	_, err := preflight.Run(ctx, preflight.Options{
		InstallerPath: filepath.Join(t.TempDir(), "grabberEM.x64.msi"),
		HasInvite:     true,
		OutputDir:     ctx.OutputDir,
		Quiet:         true,
	}, depsForScenario(scenario))
	if err != nil {
		t.Fatal(err)
	}
	testfixtures.AssertNoRawInvite(t, ctx.OutputDir, testfixtures.SecretInvite)
}
