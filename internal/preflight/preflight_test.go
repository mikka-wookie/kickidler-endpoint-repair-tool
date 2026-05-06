package preflight

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type memoryReporter struct {
	json map[string]any
	text map[string]string
	ops  []app.OperationResult
}

func (r *memoryReporter) WriteJSON(name string, v any) error {
	if r.json == nil {
		r.json = map[string]any{}
	}
	r.json[name] = v
	return nil
}

func (r *memoryReporter) WriteText(name string, content string) error {
	if r.text == nil {
		r.text = map[string]string{}
	}
	r.text[name] = content
	return nil
}

func (r *memoryReporter) WriteOperations(results []app.OperationResult) error {
	r.ops = append([]app.OperationResult(nil), results...)
	return nil
}

func (r *memoryReporter) Archive() (string, error) {
	return "", nil
}

type discardLogger struct{}

func (discardLogger) Debug(string, ...any) {}
func (discardLogger) Info(string, ...any)  {}
func (discardLogger) Warn(string, ...any)  {}
func (discardLogger) Error(string, ...any) {}
func (discardLogger) Close() error         { return nil }

func TestRequiredChecksStatusAndExitCodes(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*Dependencies, *Options)
		wantStatus string
		wantExit   int
		wantReady  bool
	}{
		{
			name:       "all required pass",
			wantStatus: StatusReady,
			wantExit:   0,
			wantReady:  true,
		},
		{
			name: "defender warning keeps repair ready",
			mutate: func(deps *Dependencies, opts *Options) {
				deps.Detect = func() (detector.DetectionReport, error) {
					report := healthyDetection()
					report.Defender.Available = false
					report.Defender.Error = "Defender status could not be read"
					return report, nil
				}
			},
			wantStatus: StatusReadyWithWarnings,
			wantExit:   1,
			wantReady:  true,
		},
		{
			name: "missing admin",
			mutate: func(deps *Dependencies, opts *Options) {
				deps.IsAdmin = func() bool { return false }
			},
			wantStatus: StatusNotReady,
			wantExit:   7,
		},
		{
			name: "missing installer",
			mutate: func(deps *Dependencies, opts *Options) {
				deps.ResolveInstaller = func(string) InstallerResolution {
					return InstallerResolution{Error: "not found"}
				}
				deps.ReadFile = func(string) error { return os.ErrNotExist }
			},
			wantStatus: StatusNotReady,
			wantExit:   7,
		},
		{
			name: "missing invite",
			mutate: func(deps *Dependencies, opts *Options) {
				opts.HasInvite = false
			},
			wantStatus: StatusNotReady,
			wantExit:   7,
		},
		{
			name: "missing powershell",
			mutate: func(deps *Dependencies, opts *Options) {
				deps.LookPath = func(name string) (string, error) {
					if strings.HasPrefix(strings.ToLower(name), "powershell") {
						return "", errors.New("not found")
					}
					return `C:\Windows\System32\` + name, nil
				}
			},
			wantStatus: StatusNotReady,
			wantExit:   7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := baseDeps(t)
			opts := baseOptions(t)
			if tt.mutate != nil {
				tt.mutate(&deps, &opts)
			}
			result := runForTest(t, opts, deps)
			if result.Status != tt.wantStatus || result.ExitCode != tt.wantExit || result.ReadyForRepair != tt.wantReady {
				t.Fatalf("got status=%s exit=%d ready=%t", result.Status, result.ExitCode, result.ReadyForRepair)
			}
		})
	}
}

func TestInstallerPathWithSpacesIsQuotedInRecommendationCommand(t *testing.T) {
	command := RepairCommand(`C:\Installers With Spaces\grabberEM.x64.msi`)
	if !strings.Contains(command, `--installer "C:\Installers With Spaces\grabberEM.x64.msi"`) {
		t.Fatalf("installer path was not quoted: %s", command)
	}
}

func TestRawInviteIsNeverSerialized(t *testing.T) {
	rawInvite := "SECRET-INVITE-VALUE"
	deps := baseDeps(t)
	opts := baseOptions(t)
	result := runForTest(t, opts, deps)
	summary := FormatSummary(result)
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	combined := string(encoded) + summary + FormatConsoleSummary(result)
	if strings.Contains(combined, rawInvite) {
		t.Fatalf("raw invite leaked into output")
	}
	if !strings.Contains(combined, "<INVITE>") {
		t.Fatalf("safe invite placeholder missing")
	}
}

func TestInstallerSHA256ComputedWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grabberEM.x64.msi")
	if err := os.WriteFile(path, []byte("installer"), 0600); err != nil {
		t.Fatal(err)
	}
	check := checkInstaller(InstallerResolution{Path: path, Provided: true}, fillDependencies(Dependencies{}))
	if check.SHA256 == "" {
		t.Fatalf("expected SHA-256 to be computed")
	}
	if !check.Exists || !check.Readable {
		t.Fatalf("expected installer to exist and be readable: %#v", check)
	}
}

func TestUnsupportedInstallerFilenameWarnsNotFatal(t *testing.T) {
	deps := baseDeps(t)
	opts := baseOptions(t)
	deps.ResolveInstaller = func(string) InstallerResolution {
		return InstallerResolution{Path: `C:\Temp\custom.msi`, Provided: true}
	}
	result := runForTest(t, opts, deps)
	if result.Status != StatusReadyWithWarnings {
		t.Fatalf("expected warning status, got %s", result.Status)
	}
	check := findCheck(result.Checks, "installer_supported_name")
	if check.Status != CheckWarning || check.Required {
		t.Fatalf("expected optional warning, got %#v", check)
	}
}

func TestFailedDetectionDoesNotPanicAndWritesPreflightResult(t *testing.T) {
	deps := baseDeps(t)
	opts := baseOptions(t)
	deps.Detect = func() (detector.DetectionReport, error) {
		return detector.DetectionReport{}, errors.New("detection failed")
	}
	reporter := &memoryReporter{}
	ctx := testContext(opts.OutputDir, reporter)
	result, err := Run(ctx, opts, deps)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusReadyWithWarnings {
		t.Fatalf("expected warning status, got %s", result.Status)
	}
	if _, ok := reporter.json["preflight-result"]; !ok {
		t.Fatalf("preflight-result was not written")
	}
}

func TestQuietJSONBehaviorIsControlledByWorkflowRunnerFlags(t *testing.T) {
	ctx := testContext(t.TempDir(), &memoryReporter{})
	ctx.Quiet = true
	ctx.JSONOutput = true
	workflow := Workflow{HasInvite: true, Deps: baseDeps(t)}
	if err := workflow.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.JSONValue == nil {
		t.Fatalf("expected JSON value to be set when quiet and json are both true")
	}

	ctx = testContext(t.TempDir(), &memoryReporter{})
	ctx.Quiet = true
	ctx.JSONOutput = false
	if err := workflow.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.JSONValue == nil {
		t.Fatalf("expected workflow result to be available even when normal stdout is suppressed")
	}
}

func baseOptions(t *testing.T) Options {
	t.Helper()
	return Options{OutputDir: t.TempDir(), HasInvite: true}
}

func baseDeps(t *testing.T) Dependencies {
	t.Helper()
	return Dependencies{
		Now:          func() time.Time { return time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC) },
		IsAdmin:      func() bool { return true },
		OS:           func() string { return "windows" },
		Architecture: func() string { return "amd64" },
		ExePath:      func() (string, error) { return `C:\Tools\kigrepair.exe`, nil },
		WorkingDir:   func() (string, error) { return `C:\Tools`, nil },
		LookPath: func(name string) (string, error) {
			return `C:\Windows\System32\` + name, nil
		},
		ResolveInstaller: func(string) InstallerResolution {
			return InstallerResolution{Path: `C:\Tools\grabberEM.x64.msi`, Discovered: true}
		},
		ReadFile:   func(string) error { return nil },
		HashFile:   func(string) (string, error) { return strings.Repeat("a", 64), nil },
		WriteProbe: func(string) error { return nil },
		Detect: func() (detector.DetectionReport, error) {
			return healthyDetection(), nil
		},
	}
}

func runForTest(t *testing.T, opts Options, deps Dependencies) Result {
	t.Helper()
	result, err := Run(testContext(opts.OutputDir, &memoryReporter{}), opts, deps)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func testContext(outputDir string, reporter app.Reporter) *app.AppContext {
	return &app.AppContext{
		OutputDir:  outputDir,
		StartedAt:  time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		Reporter:   reporter,
		Logger:     discardLogger{},
		Results:    []app.OperationResult{},
		ReportRoot: filepath.Dir(outputDir),
	}
}

func healthyDetection() detector.DetectionReport {
	return detector.DetectionReport{
		IsAdmin:                 true,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft`,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutableExists: true,
		Services: []detector.ServiceState{{
			Name:   "ngs",
			Exists: true,
			Status: "running",
		}},
		Registry: []detector.RegistryState{{
			Root: "HKLM",
			Path: `SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`,
		}},
		Defender: detector.DefenderState{
			Available:      true,
			ExclusionPaths: []string{`C:\Program Files\TeleLinkSoft`},
		},
		Health: detector.GrabberHealthHealthy,
	}
}

func findCheck(checks []CheckResult, code string) CheckResult {
	for _, check := range checks {
		if check.Code == code {
			return check
		}
	}
	return CheckResult{}
}
