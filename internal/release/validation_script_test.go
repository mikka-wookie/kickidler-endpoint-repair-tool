package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateReleaseScriptPassesValidFolder(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell release validation is Windows-focused")
	}
	dir := createTestRelease(t)
	runValidateRelease(t, dir, true)
}

func TestValidateReleaseScriptDetectsModifiedFile(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell release validation is Windows-focused")
	}
	dir := createTestRelease(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := runValidateRelease(t, dir, false)
	if !strings.Contains(out, "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got:\n%s", out)
	}
}

func TestValidateReleaseScriptRejectsBundledMSI(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell release validation is Windows-focused")
	}
	dir := createTestRelease(t)
	if err := os.WriteFile(filepath.Join(dir, "assets", "grabber.msi"), []byte("not an msi"), 0644); err != nil {
		t.Fatal(err)
	}
	out := runValidateRelease(t, dir, false)
	if !strings.Contains(out, "MSI bundled by default") {
		t.Fatalf("expected MSI rejection, got:\n%s", out)
	}
}

func TestValidateReleaseScriptRejectsSensitivePattern(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell release validation is Windows-focused")
	}
	dir := createTestRelease(t)
	if err := os.WriteFile(filepath.Join(dir, "examples", "commands.ps1"), []byte("access_token=abc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	writeChecksums(t, dir)
	out := runValidateRelease(t, dir, false)
	if !strings.Contains(out, "obvious sensitive value pattern") {
		t.Fatalf("expected sensitive pattern rejection, got:\n%s", out)
	}
}

func createTestRelease(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"kigrepair.exe":             "fake exe\n",
		"README.md":                 "# readme\n",
		"SUPPORT-RUNBOOK.md":        "# runbook\n",
		"docs/COMMAND-REFERENCE.md": "commands\n",
		"docs/SAFETY-MODEL.md":      "safety\n",
		"docs/RELEASE-CHECKLIST.md": "checklist\n",
		"docs/RELEASE-TRUST.md":     "trust\n",
		"assets/README.txt":         "no msi bundled\n",
		"examples/commands.ps1":     ".\\kigrepair.exe repair --dry-run --invite \"<INVITE>\"\n",
		"SIGNATURES.txt":            "Signing requested: false\n",
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	exeHash := fileSHA256(t, filepath.Join(dir, "kigrepair.exe"))
	manifest := fmt.Sprintf(`{
  "product": "kigrepair",
  "version": "test",
  "commit": "abc1234",
  "build_date": "2026-05-06T12:00:00Z",
  "built_by": "test@host",
  "go_version": "go1.22.0",
  "target": {"os": "windows", "arch": "amd64"},
  "checksum_algorithm": "SHA-256",
  "artifacts": [{"path": "kigrepair.exe", "type": "executable", "sha256": "%s", "size_bytes": 9, "signed": false, "signature_verified": false}],
  "security_notes": {"msi_bundled": false, "invite_included": false, "reports_included": false}
}`, exeHash)
	if err := os.WriteFile(filepath.Join(dir, "RELEASE-MANIFEST.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	writeChecksums(t, dir)
	return dir
}

func writeChecksums(t *testing.T, dir string) {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == "checksums.txt" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		lines = append(lines, fileSHA256(t, path)+"  "+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func runValidateRelease(t *testing.T, dir string, wantSuccess bool) string {
	t.Helper()
	script := filepath.Join(repoRoot(t), "scripts", "validate-release.ps1")
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-ReleaseDir", dir)
	out, err := cmd.CombinedOutput()
	if wantSuccess && err != nil {
		t.Fatalf("validate-release.ps1 failed: %v\n%s", err, out)
	}
	if !wantSuccess && err == nil {
		t.Fatalf("validate-release.ps1 unexpectedly passed:\n%s", out)
	}
	return string(out)
}
