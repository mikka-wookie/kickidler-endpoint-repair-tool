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

func TestVerifyHealthyReportSucceeds(t *testing.T) {
	got := Verify(healthyReport(), Options{})
	if got.Status != VerificationSuccess {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationSuccess, got)
	}
}

func TestVerifyMissingDefenderWarns(t *testing.T) {
	report := healthyReport()
	report.MissingDefenderPaths = []string{`C:\Program Files\TeleLinkSoft\bin`}
	got := Verify(report, Options{})
	if got.Status != VerificationWarning {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationWarning, got)
	}
}

func TestVerifyBrokenHealthFails(t *testing.T) {
	report := healthyReport()
	report.Health = detector.GrabberHealthBroken
	report.ServiceExecutableExists = false
	got := Verify(report, Options{})
	if got.Status != VerificationFailed {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationFailed, got)
	}
}

func TestVerifyInstallLogRequiredOnlyWhenInstallExecuted(t *testing.T) {
	got := Verify(healthyReport(), Options{InstallExecuted: true, MSIInstallLog: `C:\missing\msi-install.log`})
	if got.Status != VerificationFailed {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationFailed, got)
	}
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		status VerificationStatus
		want   int
	}{
		{VerificationSuccess, app.ExitSuccess},
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
	result := Verify(healthyReport(), Options{})
	result.Command = "verify"
	result.ReportDir = `C:\ProgramData\kigrepair\Reports\test`
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
			return healthyReport()
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
	result := Verify(healthyReport(), Options{})
	result.Command = "verify"
	result.ExitCode = ExitCode(result.Status)
	result.ReportDir = `C:\ProgramData\kigrepair\Reports\test`
	result.Health = string(result.Detection.Health)
	result.InstallMode = string(result.Detection.InstallMode)
	result.InstallRoot = result.Detection.InstallRoot
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"command", "status", "exit_code", "report_dir", "health", "install_mode", "install_root", "detection"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("JSON result missing key %q: %s", key, string(data))
		}
	}
}

func healthyReport() detector.DetectionReport {
	return detector.DetectionReport{
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft\bin`,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutableExists: true,
		RequiredDefenderPaths:   []string{`C:\Program Files\TeleLinkSoft\bin`},
		Services: []detector.ServiceState{
			{Name: "ngs", Exists: true, Status: "running"},
		},
		Processes: []detector.ProcessState{
			{Name: "grabber2.exe", PID: 100, ExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, MatchedByName: true},
		},
		Defender: detector.DefenderState{Available: true},
	}
}
