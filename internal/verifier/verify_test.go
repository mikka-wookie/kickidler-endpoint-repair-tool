package verifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/reports"
)

func TestVerifyInstallation(t *testing.T) {
	tests := []struct {
		name string
		edit func(*detector.DetectionReport, string)
		opts VerifyOptions
		want VerificationStatus
	}{
		{
			name: "healthy standard install",
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationPassed,
		},
		{
			name: "healthy with missing defender",
			edit: func(report *detector.DetectionReport, root string) {
				report.MissingDefenderPaths = []string{root}
				report.Defender.MissingPaths = []string{root}
				report.Defender.ExclusionPaths = nil
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationWarning,
		},
		{
			name: "defender unavailable",
			edit: func(report *detector.DetectionReport, root string) {
				report.Defender.Available = false
				report.Defender.Error = "unavailable"
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationWarning,
		},
		{
			name: "service stopped but executable exists",
			edit: func(report *detector.DetectionReport, root string) {
				report.Services[0].Status = "stopped"
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowStoppedServiceWarning: true, AllowDefenderUnavailable: true},
			want: VerificationWarning,
		},
		{
			name: "executable missing",
			edit: func(report *detector.DetectionReport, root string) {
				report.ServiceExecutableExists = false
				report.ServiceExecutablePath = filepath.Join(root, "missing.exe")
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationFailed,
		},
		{
			name: "not installed after install",
			edit: func(report *detector.DetectionReport, root string) {
				*report = detector.DetectionReport{
					Health:      detector.GrabberHealthNotInstalled,
					InstallMode: detector.InstallModeNotInstalled,
					Defender:    detector.DefenderState{Available: true},
				}
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationFailed,
		},
		{
			name: "process missing but service running",
			edit: func(report *detector.DetectionReport, root string) {
				report.Processes = nil
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationWarning,
		},
		{
			name: "msi log missing when install executed",
			opts: VerifyOptions{InstallExecuted: true, MSIInstallLogPath: filepath.Join(t.TempDir(), "missing.log"), RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationWarning,
		},
		{
			name: "hidden wmi healthy",
			edit: func(report *detector.DetectionReport, root string) {
				systemRoot := filepath.Join(t.TempDir(), "Windows")
				wmiRoot := filepath.Join(systemRoot, "System32", "wmi")
				bin := filepath.Join(wmiRoot, "bin")
				exe := tempFile(t, bin, "svchost.exe")
				*report = healthyReport(wmiRoot, exe)
				report.System.SystemRoot = systemRoot
				report.InstallMode = detector.InstallModeHiddenWMI
				report.InstallRoot = wmiRoot
				report.BinaryDir = bin
				report.PrimaryService = "WmiProviderSE"
				report.PrimaryServiceImagePath = exe
				report.ServiceExecutablePath = exe
				report.RequiredDefenderPaths = []string{wmiRoot}
				report.Defender.RequiredPaths = []string{wmiRoot}
				report.Defender.ExclusionPaths = []string{wmiRoot}
				report.Services = []detector.ServiceState{{Name: "WmiProviderSE", Exists: true, Status: "running", ImagePath: exe}}
				report.Processes = []detector.ProcessState{{Name: "svchost.exe", ExecutablePath: exe, MatchedByExactPath: true}}
			},
			opts: VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true},
			want: VerificationPassed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			exe := tempFile(t, root, "grabber2.exe")
			report := healthyReport(root, exe)
			if tt.edit != nil {
				tt.edit(&report, root)
			}
			got := VerifyInstallation(report, tt.opts)
			if got.OverallStatus != tt.want {
				t.Fatalf("OverallStatus = %s, want %s: %#v", got.OverallStatus, tt.want, got)
			}
		})
	}
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		status VerificationStatus
		want   int
	}{
		{VerificationPassed, app.ExitSuccess},
		{VerificationWarning, app.ExitWarnings},
		{VerificationFailed, app.ExitVerificationFailed},
	}
	for _, tt := range tests {
		if got := ExitCode(tt.status); got != tt.want {
			t.Fatalf("ExitCode(%s) = %d, want %d", tt.status, got, tt.want)
		}
	}
}

func TestFormatSummaryIncludesSupportAction(t *testing.T) {
	result := VerifyInstallation(healthyReport(`C:\Program Files\TeleLinkSoft\bin`, `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`), VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true})
	result.Command = "verify"
	result.ReportDir = `C:\ProgramData\kigrepair\Reports\test`
	result.ExitCode = ExitCode(result.OverallStatus)
	summary := FormatSummary(result)
	for _, want := range []string{
		"Kigrepair Verify",
		"Expected process running: yes",
		"Next recommended support action:",
		"No repair required.",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q:\n%s", want, summary)
		}
	}
}

func TestVerifyWorkflowWritesExpectedReportsAndOperations(t *testing.T) {
	dir := t.TempDir()
	reporter, err := reports.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	detectCalls := 0
	ctx := app.NewContext()
	ctx.OutputDir = dir
	ctx.Reporter = reporter
	ctx.Logger = logging.Discard()

	err = VerifyWorkflow{
		Detect: func() detector.DetectionReport {
			detectCalls++
			return healthyReport(`C:\Program Files\TeleLinkSoft\bin`, `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`)
		},
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if detectCalls != 1 {
		t.Fatalf("detect calls = %d, want 1", detectCalls)
	}
	if ctx.ExitCode != app.ExitSuccess {
		t.Fatalf("exit code = %d, want %d", ctx.ExitCode, app.ExitSuccess)
	}
	for _, name := range []string{"initial-detection.json", "verification-result.json", "operations.json", "summary.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
	if len(ctx.Results) != 2 {
		t.Fatalf("operation count = %d, want 2: %#v", len(ctx.Results), ctx.Results)
	}
	if ctx.Results[0].Step != "verify.detection" || ctx.Results[1].Step != "verify.final" {
		t.Fatalf("operation steps = %#v", ctx.Results)
	}
}

func TestVerificationResultJSONShapeHasTopLevelFields(t *testing.T) {
	result := VerifyInstallation(healthyReport(`C:\Program Files\TeleLinkSoft\bin`, `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`), VerifyOptions{RequireRunningProcess: true, AllowDefenderUnavailable: true})
	result.Command = "verify"
	result.ExitCode = ExitCode(result.OverallStatus)
	result.ReportDir = `C:\ProgramData\kigrepair\Reports\test`
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"command", "overall_status", "exit_code", "report_dir", "health", "install_mode", "install_root", "detection"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("JSON result missing key %q: %s", key, string(data))
		}
	}
}

func healthyReport(root string, exe string) detector.DetectionReport {
	return detector.DetectionReport{
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             root,
		PrimaryService:          "ngs",
		PrimaryServiceImagePath: exe,
		ServiceExecutablePath:   exe,
		ServiceExecutableExists: true,
		RequiredDefenderPaths:   []string{root},
		Services:                []detector.ServiceState{{Name: "ngs", Exists: true, Status: "running", ImagePath: exe}},
		Processes:               []detector.ProcessState{{Name: "grabber2.exe", ExecutablePath: exe, MatchedByName: true}},
		Defender: detector.DefenderState{
			Available:      true,
			RequiredPaths:  []string{root},
			ExclusionPaths: []string{root},
			CoveredPaths:   []string{root},
		},
	}
}

func tempFile(t *testing.T, dir string, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("exe"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
