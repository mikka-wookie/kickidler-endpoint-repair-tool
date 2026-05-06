package defender

import (
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/reports"
)

type testLogger struct{}

func (testLogger) Debug(string, ...any) {}
func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) Close() error         { return nil }

type fakeAdder struct {
	result CommandResult
	added  []string
}

func (f *fakeAdder) AddExclusion(path string) CommandResult {
	f.added = append(f.added, path)
	return f.result
}

func TestRequiredPaths(t *testing.T) {
	system := detector.SystemState{SystemRoot: `C:\Windows`}
	tests := []struct {
		name         string
		report       detector.DetectionReport
		want         []string
		wantWarnings bool
	}{
		{
			name:   "standard mode uses install root",
			report: detector.DetectionReport{System: system, InstallMode: detector.InstallModeStandard, InstallRoot: `C:\Program Files\TeleLinkSoft\bin`},
			want:   []string{`C:\Program Files\TeleLinkSoft\bin`},
		},
		{
			name:   "helper mode uses install root",
			report: detector.DetectionReport{System: system, InstallMode: detector.InstallModeHelper, InstallRoot: `C:\Program Files\TeleLinkSoftHelper`},
			want:   []string{`C:\Program Files\TeleLinkSoftHelper`},
		},
		{
			name:   "hidden wmi uses wmi root",
			report: detector.DetectionReport{System: system, InstallMode: detector.InstallModeHiddenWMI, InstallRoot: `C:\Windows\System32\wmi\bin`},
			want:   []string{`C:\Windows\System32\wmi`},
		},
		{
			name:         "not installed returns warning and no path",
			report:       detector.DetectionReport{System: system, InstallMode: detector.InstallModeNotInstalled},
			wantWarnings: true,
		},
		{
			name:         "unknown returns warning and no path",
			report:       detector.DetectionReport{System: system, InstallMode: detector.InstallModeUnknown},
			wantWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warnings := RequiredPaths(tt.report, false)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("RequiredPaths() = %#v, want %#v", got, tt.want)
			}
			if (len(warnings) > 0) != tt.wantWarnings {
				t.Fatalf("warnings = %#v, want warnings %v", warnings, tt.wantWarnings)
			}
		})
	}
}

func TestCoveragePlanning(t *testing.T) {
	required := []string{`C:\Program Files\TeleLinkSoft\bin`}
	tests := []struct {
		name       string
		exclusions []string
		wantMiss   []string
	}{
		{
			name:       "existing exact exclusion",
			exclusions: []string{`C:\Program Files\TeleLinkSoft\bin`},
		},
		{
			name:       "parent exclusion covers child",
			exclusions: []string{`C:\Program Files\TeleLinkSoft`},
		},
		{
			name:     "missing path",
			wantMiss: []string{`C:\Program Files\TeleLinkSoft\bin`},
		},
		{
			name:       "child exclusion does not cover parent",
			exclusions: []string{`C:\Program Files\TeleLinkSoft\bin\child`},
			wantMiss:   []string{`C:\Program Files\TeleLinkSoft\bin`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingPaths(required, tt.exclusions)
			if strings.Join(got, "|") != strings.Join(tt.wantMiss, "|") {
				t.Fatalf("missingPaths() = %#v, want %#v", got, tt.wantMiss)
			}
		})
	}
}

func TestConfirmationBehavior(t *testing.T) {
	ctx := &app.AppContext{NonInteractive: true}
	if err := ConfirmEnsure(ctx, false, strings.NewReader(""), nil); err == nil {
		t.Fatal("expected confirmation required")
	}
	if err := ConfirmEnsure(ctx, true, strings.NewReader(""), nil); err != nil {
		t.Fatalf("expected --yes to proceed: %v", err)
	}
	statusCtx := newWorkflowContext(t)
	adder := &fakeAdder{result: CommandResult{ExitCode: 0}}
	report := standardReport(nil)
	workflow := DefenderWorkflow{Ensure: false, Yes: false, Adder: adder, Detect: func() detector.DetectionReport { return report }}
	if err := workflow.Run(statusCtx); err != nil {
		t.Fatal(err)
	}
	if len(adder.added) != 0 {
		t.Fatalf("status-only attempted adds: %#v", adder.added)
	}
}

func TestWorkflowExitClassification(t *testing.T) {
	tests := []struct {
		name       string
		ensure     bool
		yes        bool
		admin      bool
		initial    detector.DetectionReport
		final      detector.DetectionReport
		addResult  CommandResult
		wantCode   int
		wantAdds   int
		wantOutput string
	}{
		{
			name:     "already covered",
			ensure:   true,
			yes:      true,
			admin:    true,
			initial:  standardReport([]string{`C:\Program Files\TeleLinkSoft`}),
			final:    standardReport([]string{`C:\Program Files\TeleLinkSoft`}),
			wantCode: ExitOK,
		},
		{
			name:     "missing status only",
			initial:  standardReport(nil),
			final:    standardReport(nil),
			wantCode: ExitWarnings,
		},
		{
			name:     "ensure requires admin",
			ensure:   true,
			yes:      true,
			admin:    false,
			initial:  standardReport([]string{`C:\Program Files\TeleLinkSoft`}),
			final:    standardReport([]string{`C:\Program Files\TeleLinkSoft`}),
			wantCode: ExitAdminRequired,
		},
		{
			name:      "add failed",
			ensure:    true,
			yes:       true,
			admin:     true,
			initial:   standardReport(nil),
			final:     standardReport(nil),
			addResult: CommandResult{ExitCode: 1, Output: "failed"},
			wantCode:  ExitAddFailed,
			wantAdds:  1,
		},
		{
			name:      "verification failed",
			ensure:    true,
			yes:       true,
			admin:     true,
			initial:   standardReport(nil),
			final:     standardReport(nil),
			addResult: CommandResult{ExitCode: 0},
			wantCode:  ExitVerificationFailed,
			wantAdds:  1,
		},
		{
			name:      "add succeeds and verifies",
			ensure:    true,
			yes:       true,
			admin:     true,
			initial:   standardReport(nil),
			final:     standardReport([]string{`C:\Program Files\TeleLinkSoft\bin`}),
			addResult: CommandResult{ExitCode: 0},
			wantCode:  ExitOK,
			wantAdds:  1,
		},
		{
			name:       "non-interactive ensure missing yes",
			ensure:     true,
			initial:    standardReport(nil),
			final:      standardReport(nil),
			wantCode:   ExitConfirmationRequired,
			wantAdds:   0,
			wantOutput: "Defender ensure in non-interactive mode requires --yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newWorkflowContext(t)
			if tt.name == "non-interactive ensure missing yes" {
				ctx.NonInteractive = true
			}
			adder := &fakeAdder{result: tt.addResult}
			calls := 0
			workflow := DefenderWorkflow{
				Ensure:  tt.ensure,
				Yes:     tt.yes,
				Adder:   adder,
				IsAdmin: func() bool { return tt.admin },
				Detect: func() detector.DetectionReport {
					calls++
					if calls == 1 {
						return tt.initial
					}
					return tt.final
				},
			}
			if err := workflow.Run(ctx); err != nil {
				t.Fatal(err)
			}
			if ctx.ExitCode != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", ctx.ExitCode, tt.wantCode)
			}
			if len(adder.added) != tt.wantAdds {
				t.Fatalf("adds = %#v, want %d", adder.added, tt.wantAdds)
			}
			if tt.wantOutput != "" {
				result, ok := ctx.JSONValue.(DefenderEnsureResult)
				if !ok {
					t.Fatalf("JSONValue type = %T", ctx.JSONValue)
				}
				if !contains(result.Errors, tt.wantOutput) {
					t.Fatalf("errors = %#v, want %q", result.Errors, tt.wantOutput)
				}
			}
		})
	}
}

func newWorkflowContext(t *testing.T) *app.AppContext {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "report")
	reporter, err := reports.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := app.NewContext()
	ctx.OutputDir = dir
	ctx.Quiet = true
	ctx.Reporter = reporter
	ctx.Logger = testLogger{}
	return ctx
}

func standardReport(exclusions []string) detector.DetectionReport {
	report := detector.DetectionReport{
		IsAdmin:     true,
		System:      detector.SystemState{SystemRoot: `C:\Windows`},
		InstallMode: detector.InstallModeStandard,
		InstallRoot: `C:\Program Files\TeleLinkSoft\bin`,
		Defender: detector.DefenderState{
			Available:      true,
			ExclusionPaths: exclusions,
		},
	}
	_, report.Defender.CoveredPaths, report.Defender.MissingPaths = detector.EvaluateDefenderCoverage([]string{report.InstallRoot}, exclusions)
	return report
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
