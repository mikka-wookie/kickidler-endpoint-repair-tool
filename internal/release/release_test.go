package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseFilesExist(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"scripts/build-release.ps1",
		"README.md",
		"SUPPORT-RUNBOOK.md",
		"assets/README.txt",
		"examples/commands.ps1",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s missing: %v", rel, err)
		}
	}
}

func TestReadmeContainsRequiredSafetySections(t *testing.T) {
	readme := readText(t, "README.md")
	required := []string{
		"check",
		"verify",
		"preflight",
		"repair --dry-run",
		"cleanup --dry-run",
		"collect-report",
		"reports list",
		"defender --ensure",
		"reports cleanup --yes",
		`C:\ProgramData\kigrepair\Reports\<timestamp>\`,
		"<INVITE>",
	}
	for _, value := range required {
		if !strings.Contains(readme, value) {
			t.Fatalf("README.md missing %q", value)
		}
	}
}

func TestCommandExamplesDoNotContainRealInvite(t *testing.T) {
	for _, rel := range []string{"examples/commands.ps1", "README.md", "SUPPORT-RUNBOOK.md", "assets/README.txt"} {
		content := readText(t, rel)
		if strings.Contains(content, "SECRET") || strings.Contains(content, "invite=secret") {
			t.Fatalf("%s contains a secret-like invite example", rel)
		}
		if strings.Contains(content, "--invite") && !strings.Contains(content, "<INVITE>") {
			t.Fatalf("%s contains --invite without placeholder", rel)
		}
	}
}

func TestReleaseScriptInjectsVersionPackageMetadata(t *testing.T) {
	content := readText(t, "scripts/build-release.ps1")
	for _, value := range []string{
		"kigrepair/internal/version.Version",
		"kigrepair/internal/version.Commit",
		"kigrepair/internal/version.BuildDate",
		"kigrepair/internal/version.BuiltBy",
		"go test ./...",
		"checksums.txt",
		"Compress-Archive",
	} {
		if !strings.Contains(content, value) {
			t.Fatalf("build-release.ps1 missing %q", value)
		}
	}
}

func TestReleaseScriptDoesNotReferenceSecretEnvironmentNames(t *testing.T) {
	content := strings.ToLower(readText(t, "scripts/build-release.ps1"))
	for _, value := range []string{"token", "secret", "password", "invite"} {
		if strings.Contains(content, value) {
			t.Fatalf("build-release.ps1 references secret-like value %q", value)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func readText(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
