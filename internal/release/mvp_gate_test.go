package release

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseBlockerScriptSeverityRules(t *testing.T) {
	cases := []struct {
		name     string
		severity string
		status   string
		wantFail bool
	}{
		{name: "open P0 fails", severity: "P0", status: "open", wantFail: true},
		{name: "open P1 fails", severity: "P1", status: "open", wantFail: true},
		{name: "open P2 warns but passes", severity: "P2", status: "open", wantFail: false},
		{name: "closed P0 passes", severity: "P0", status: "closed", wantFail: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			releaseBlockers := filepath.Join(dir, "RELEASE-BLOCKERS.md")
			bugBurndown := filepath.Join(dir, "BUG-BURNDOWN.md")
			blockersJSON := filepath.Join(dir, "release-blockers.json")
			writeFile(t, releaseBlockers, "# Release Blockers\n")
			writeFile(t, bugBurndown, "| ID | Severity | Area | Title | Status | Repro steps | Expected | Actual | Owner | Fixed in | Regression test |\n|---|---|---|---|---|---|---|---|---|---|---|\n| BUG-TEMP-001 | "+tc.severity+" | test | temporary | "+tc.status+" | n/a | n/a | n/a | qa | n/a | n/a |\n")
			writeFile(t, blockersJSON, `{"blockers":[]}`)

			_, err := runPowerShell(t,
				filepath.Join(repoRoot(t), "scripts", "check-release-blockers.ps1"),
				"-ReleaseBlockers", releaseBlockers,
				"-BugBurndown", bugBurndown,
				"-BlockersJson", blockersJSON,
			)
			if tc.wantFail && err == nil {
				t.Fatal("expected blocker script to fail")
			}
			if !tc.wantFail && err != nil {
				t.Fatalf("expected blocker script to pass: %v", err)
			}
		})
	}
}

func TestMVPReadinessDecisionRules(t *testing.T) {
	cases := []struct {
		name       string
		bugRows    string
		smokeState string
		want       string
	}{
		{name: "P0 no go", bugRows: bugRow("BUG-1", "P0", "open"), smokeState: "passed", want: "Release recommendation: NO-GO"},
		{name: "P1 no go", bugRows: bugRow("BUG-1", "P1", "open"), smokeState: "passed", want: "Release recommendation: NO-GO"},
		{name: "smoke fail no go", bugRows: "", smokeState: "failed", want: "Release recommendation: NO-GO"},
		{name: "clean smoke pass pilot candidate", bugRows: bugRow("BUG-1", "P0", "closed"), smokeState: "passed", want: "Release recommendation: PILOT CANDIDATE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			bugs := filepath.Join(dir, "BUG-BURNDOWN.md")
			smoke := filepath.Join(dir, "smoke-result.json")
			out := filepath.Join(dir, "MVP-READINESS.md")
			writeFile(t, bugs, bugHeader()+tc.bugRows)
			writeJSON(t, smoke, map[string]any{"status": tc.smokeState, "commands": []any{}})

			_, err := runPowerShell(t,
				filepath.Join(repoRoot(t), "scripts", "generate-mvp-readiness.ps1"),
				"-SmokeResult", smoke,
				"-BugBurndown", bugs,
				"-OutFile", out,
			)
			if err != nil {
				t.Fatalf("readiness generator failed: %v", err)
			}
			content := readLocal(t, out)
			if !strings.Contains(content, tc.want) {
				t.Fatalf("readiness missing %q:\n%s", tc.want, content)
			}
		})
	}
}

func TestSmokeResultModelAndFailureClassification(t *testing.T) {
	dir := t.TempDir()
	releaseBlockers := filepath.Join(dir, "RELEASE-BLOCKERS.md")
	bugBurndown := filepath.Join(dir, "BUG-BURNDOWN.md")
	blockersJSON := filepath.Join(dir, "release-blockers.json")
	smoke := filepath.Join(dir, "smoke-result.json")
	writeFile(t, releaseBlockers, "# Release Blockers\n")
	writeFile(t, bugBurndown, bugHeader())
	writeFile(t, blockersJSON, `{"blockers":[]}`)
	writeJSON(t, smoke, map[string]any{
		"status": "failed",
		"commands": []map[string]any{{
			"name":             "check",
			"redacted_command": ".\\kigrepair.exe check",
			"exit_code":        2,
			"failure_severity": "P1",
			"report_paths":     []string{`C:\ProgramData\kigrepair\Reports\run`},
		}},
	})
	content := readLocal(t, smoke)
	if strings.Contains(content, "REAL-SECRET-INVITE") {
		t.Fatal("smoke fixture leaked invite")
	}
	var parsed struct {
		Commands []struct {
			ExitCode    int      `json:"exit_code"`
			ReportPaths []string `json:"report_paths"`
		} `json:"commands"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("smoke JSON invalid: %v", err)
	}
	if len(parsed.Commands) != 1 || parsed.Commands[0].ExitCode != 2 || len(parsed.Commands[0].ReportPaths) != 1 {
		t.Fatalf("smoke command fields not captured: %#v", parsed.Commands)
	}
	_, err := runPowerShell(t,
		filepath.Join(repoRoot(t), "scripts", "check-release-blockers.ps1"),
		"-ReleaseBlockers", releaseBlockers,
		"-BugBurndown", bugBurndown,
		"-BlockersJson", blockersJSON,
		"-SmokeResult", smoke,
	)
	if err == nil {
		t.Fatal("failed P1 smoke result should block release")
	}
}

func TestRedactionScriptRules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "allowed.txt"), "Use <INVITE> or <REDACTED> placeholders only.\n")
	if _, err := runPowerShell(t, filepath.Join(repoRoot(t), "scripts", "check-redaction.ps1"), "-Path", dir, "-SecretPattern", "<INVITE>"); err != nil {
		t.Fatalf("placeholders should pass: %v", err)
	}

	writeFile(t, filepath.Join(dir, "leak.txt"), "invite=REAL-SECRET-INVITE\n")
	if _, err := runPowerShell(t, filepath.Join(repoRoot(t), "scripts", "check-redaction.ps1"), "-Path", dir, "-SecretPattern", "REAL-SECRET-INVITE"); err == nil {
		t.Fatal("fake invite should be detected")
	}

	dir2 := t.TempDir()
	writeFile(t, filepath.Join(dir2, "token.txt"), "Authorization: Bearer abcdefghijklmnopqrstuvwxyz\n")
	if _, err := runPowerShell(t, filepath.Join(repoRoot(t), "scripts", "check-redaction.ps1"), "-Path", dir2); err == nil {
		t.Fatal("raw bearer token pattern should be detected")
	}
}

func TestStep43DocsAndScriptsExist(t *testing.T) {
	for _, rel := range []string{
		"docs/BUG-BURNDOWN.md",
		"docs/MVP-READINESS.md",
		"docs/REGRESSION-TEST-PLAN.md",
		"docs/RELEASE-BLOCKERS.md",
		"docs/KNOWN-LIMITATIONS.md",
		"docs/GUI-SMOKE-CHECKLIST.md",
		"scripts/smoke-mvp.ps1",
		"scripts/generate-mvp-readiness.ps1",
		"scripts/check-release-blockers.ps1",
		"scripts/check-redaction.ps1",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s missing: %v", rel, err)
		}
	}
}

func TestBuildScriptKeepsWindowsGuiFlagGUIOnly(t *testing.T) {
	content := readText(t, "scripts/build-release.ps1")
	guiLine := `go build -trimpath -ldflags "$ldflags -H windowsgui" -o $guiPath ./cmd/kigrepair-gui`
	cliLine := `go build -trimpath -ldflags $ldflags -o $cliPath ./cmd/kigrepair`
	if !strings.Contains(content, guiLine) {
		t.Fatal("GUI build must use -H windowsgui")
	}
	if !strings.Contains(content, cliLine) {
		t.Fatal("CLI build command not found")
	}
	if strings.Contains(cliLine, "-H windowsgui") {
		t.Fatal("CLI build must not use -H windowsgui")
	}
}

func runPowerShell(t *testing.T, script string, args ...string) (string, error) {
	t.Helper()
	shell := findPowerShell(t)
	allArgs := []string{"-NoProfile"}
	if filepath.Base(shell) == "powershell.exe" || filepath.Base(shell) == "powershell" {
		allArgs = append(allArgs, "-ExecutionPolicy", "Bypass")
	}
	allArgs = append(allArgs, "-File", script)
	allArgs = append(allArgs, args...)
	cmd := exec.Command(shell, allArgs...)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func findPowerShell(t *testing.T) string {
	t.Helper()
	candidates := []string{"powershell.exe", "powershell", "pwsh.exe", "pwsh"}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell is not available")
	}
	t.Fatal("PowerShell is not available")
	return ""
}

func bugHeader() string {
	return "| ID | Severity | Area | Title | Status | Repro steps | Expected | Actual | Owner | Fixed in | Regression test |\n|---|---|---|---|---|---|---|---|---|---|---|\n"
}

func bugRow(id string, severity string, status string) string {
	return "| " + id + " | " + severity + " | test | temporary | " + status + " | n/a | n/a | n/a | qa | n/a | n/a |\n"
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}

func readLocal(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
