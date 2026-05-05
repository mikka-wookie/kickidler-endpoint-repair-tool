package reports

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSupportBundleUsesDeterministicLayoutAndManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte("summary invite=SECRET123"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "system")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "environment.json"), []byte(`{"invite":"SECRET123"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installer-validation.json"), []byte(`{"status":"valid"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rollback-info.json"), []byte(`{"has_invite":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CreateSupportBundle(dir)
	if err != nil {
		t.Fatalf("CreateSupportBundle() error = %v", err)
	}
	if result.Path != filepath.Join(dir, SupportBundleName) {
		t.Fatalf("bundle path = %q", result.Path)
	}

	reader, err := zip.OpenReader(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	names := map[string]bool{}
	contents := map[string]string{}
	for _, file := range reader.File {
		names[file.Name] = true
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		contents[file.Name] = string(data)
	}
	if names["kigrepair-support-bundle/"+SupportBundleName] {
		t.Fatalf("bundle included itself")
	}
	if !names["kigrepair-support-bundle/manifest.json"] {
		t.Fatalf("manifest.json was not included")
	}
	if !names["kigrepair-support-bundle/summary.txt"] {
		t.Fatalf("summary.txt was not included")
	}
	if !names["kigrepair-support-bundle/system/environment.json"] {
		t.Fatalf("system/environment.json was not included")
	}
	if !names["kigrepair-support-bundle/reports/installer-validation.json"] {
		t.Fatalf("installer-validation.json was not included")
	}
	if !names["kigrepair-support-bundle/reports/rollback-info.json"] {
		t.Fatalf("rollback-info.json was not included")
	}
	if strings.Contains(contents["kigrepair-support-bundle/summary.txt"], "SECRET123") {
		t.Fatalf("summary was not redacted: %q", contents["kigrepair-support-bundle/summary.txt"])
	}
	if strings.Contains(contents["kigrepair-support-bundle/system/environment.json"], "SECRET123") {
		t.Fatalf("environment was not redacted: %q", contents["kigrepair-support-bundle/system/environment.json"])
	}

	var manifest BundleManifest
	if err := json.Unmarshal([]byte(contents["kigrepair-support-bundle/manifest.json"]), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 4 {
		t.Fatalf("manifest files = %d, want 4", len(manifest.Files))
	}
	for _, file := range manifest.Files {
		if file.SHA256 == "" {
			t.Fatalf("manifest file missing sha256: %#v", file)
		}
	}
}

func TestCleanArchivePathRejectsUnsafePaths(t *testing.T) {
	rejected := []string{"../evil.txt", "/absolute.txt", `C:\absolute.txt`, "reports/../evil.txt"}
	for _, input := range rejected {
		if _, err := cleanArchivePath(input); err == nil {
			t.Fatalf("cleanArchivePath(%q) succeeded, want error", input)
		}
	}
	got, err := cleanArchivePath("reports/initial-detection.json")
	if err != nil {
		t.Fatalf("cleanArchivePath() error = %v", err)
	}
	if got != "reports/initial-detection.json" {
		t.Fatalf("cleanArchivePath() = %q", got)
	}
}

func TestCreateSupportBundleMissingOptionalFilesWarns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := CreateSupportBundle(dir)
	if err != nil {
		t.Fatalf("CreateSupportBundle() error = %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warnings for missing optional files")
	}
	if len(result.Included) != 1 || result.Included[0].Path != "summary.txt" {
		t.Fatalf("included files = %#v", result.Included)
	}
}
