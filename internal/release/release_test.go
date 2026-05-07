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
		"scripts/validate-release.ps1",
		"README.md",
		"SUPPORT-RUNBOOK.md",
		"assets/README.txt",
		"examples/commands.ps1",
		"docs/RELEASE-TRUST.md",
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
		"RELEASE-MANIFEST.json",
		"SIGNATURES.txt",
		"signtool",
		"Compress-Archive",
	} {
		if !strings.Contains(content, value) {
			t.Fatalf("build-release.ps1 missing %q", value)
		}
	}
}

func TestReleaseScriptBuildsGUIWithoutConsoleWindow(t *testing.T) {
	content := readText(t, "scripts/build-release.ps1")
	if !strings.Contains(content, "-H windowsgui") {
		t.Fatal("build-release.ps1 must build kigrepair-gui.exe with -H windowsgui")
	}
}

func TestReleaseScriptDoesNotReferenceSecretEnvironmentNames(t *testing.T) {
	content := strings.ToLower(readText(t, "scripts/build-release.ps1"))
	for _, value := range []string{"access_token", "refresh_token", "authorization: bearer", "password="} {
		if strings.Contains(content, value) {
			t.Fatalf("build-release.ps1 references sensitive value pattern %q", value)
		}
	}
}

func TestReleaseScriptSupportsSigningPolicy(t *testing.T) {
	content := readText(t, "scripts/build-release.ps1")
	required := []string{
		"[switch]$Sign",
		"[string]$CertificateThumbprint",
		"-Sign requires -CertificateThumbprint",
		"signtool sign /fd SHA256 /tr <TIMESTAMP_URL> /td SHA256 /sha1 <THUMBPRINT>",
		"signtool verify /pa /v",
		"Signing: not requested",
	}
	for _, value := range required {
		if !strings.Contains(content, value) {
			t.Fatalf("build-release.ps1 missing signing policy %q", value)
		}
	}
}

func TestReleaseManifestTemplateContainsRequiredTrustFields(t *testing.T) {
	content := readText(t, "scripts/build-release.ps1")
	required := []string{
		"product = \"kigrepair\"",
		"checksum_algorithm = \"SHA-256\"",
		"msi_bundled = $false",
		"invite_included = $false",
		"reports_included = $false",
		"artifacts = $Artifacts",
		"zip = $Zip",
	}
	for _, value := range required {
		if !strings.Contains(content, value) {
			t.Fatalf("manifest generation missing %q", value)
		}
	}
}

func TestValidateReleaseScriptCoversRequiredFailures(t *testing.T) {
	content := readText(t, "scripts/validate-release.ps1")
	required := []string{
		"checksum mismatch",
		"MSI bundled by default",
		"RELEASE-MANIFEST.json is not valid JSON",
		"required signed artifact is unsigned or unverified",
		"obvious sensitive value pattern",
		"zip checksum mismatch",
	}
	for _, value := range required {
		if !strings.Contains(content, value) {
			t.Fatalf("validate-release.ps1 missing validation %q", value)
		}
	}
}

func TestReleaseDocsDescribeTrustChain(t *testing.T) {
	content := readText(t, "docs/RELEASE-TRUST.md")
	for _, value := range []string{"checksums.txt", "RELEASE-MANIFEST.json", "SIGNATURES.txt", "Get-AuthenticodeSignature", "MSI", "<version>"} {
		if !strings.Contains(content, value) {
			t.Fatalf("RELEASE-TRUST.md missing %q", value)
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
