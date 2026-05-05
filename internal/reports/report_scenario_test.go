package reports_test

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/reports"
	"kigrepair/internal/testfixtures"
)

func TestReportCleanupSafetyScenario(t *testing.T) {
	root := t.TempDir()
	active := makeReportDir(t, root, "2026-05-05_12-00-00")
	old := makeReportDir(t, root, "2026-03-01_12-00-00")
	kept := makeReportDir(t, root, "2026-05-04_12-00-00")
	nonReport := filepath.Join(root, "old-not-report")
	if err := os.MkdirAll(nonReport, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonReport, "random.txt"), []byte("not a report"), 0644); err != nil {
		t.Fatal(err)
	}

	plan, err := reports.BuildReportCleanupPlan(root, reports.ReportCleanupOptions{
		OlderThan:       "30d",
		KeepLast:        1,
		DryRun:          true,
		ActiveReportDir: active,
		Now:             "2026-05-05T12:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.PlannedDeletes) != 1 || !samePath(plan.PlannedDeletes[0].Path, old) {
		t.Fatalf("planned deletes = %#v, want only %s", plan.PlannedDeletes, old)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatalf("dry-run deleted old report: %v", err)
	}
	result := reports.ExecuteReportCleanupPlan(root, plan, true)
	if result.DeletedCount != 0 && len(result.Deleted) != 0 {
		t.Fatalf("dry-run should not delete: %#v", result)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatalf("dry-run removed old report: %v", err)
	}

	realPlan := plan
	realPlan.DryRun = false
	result = reports.ExecuteReportCleanupPlan(root, realPlan, true)
	if result.Status != "success" {
		t.Fatalf("cleanup result = %#v", result)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old report still exists or unexpected stat error: %v", err)
	}
	for _, path := range []string{active, kept, nonReport} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s should remain: %v", path, err)
		}
	}
	if err := reports.ValidateReportDeleteTarget(root, filepath.Join(root, "..", filepath.Base(old)), active); err == nil {
		t.Fatal("path traversal candidate was accepted")
	}
}

func TestSupportBundleRedactsInviteAndHasManifestHashes(t *testing.T) {
	dir := t.TempDir()
	secret := testfixtures.SecretInvite
	writeFile(t, filepath.Join(dir, "summary.txt"), "invite="+secret+"\nAuthorization: Bearer abc\n")
	writeFile(t, filepath.Join(dir, "repair.log"), `"invite":"`+secret+`"`+"\n")
	writeFile(t, filepath.Join(dir, "operations.json"), `[{"step":"install","message":"invite=`+secret+`"}]`)

	archive, err := reports.CreateSupportBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	testfixtures.AssertZipNoRawInvite(t, archive.Path, secret)
	testfixtures.AssertZipNoRawInvite(t, archive.Path, "Bearer abc")
	source, err := os.ReadFile(filepath.Join(dir, "summary.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), secret) {
		t.Fatal("source report was overwritten; bundle creation should redact zip entries only")
	}
	assertManifestHashes(t, archive.Path)
}

func makeReportDir(t *testing.T, root string, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, "summary.txt"), "Status: success\n")
	parsed, err := time.Parse("2006-01-02_15-04-05", name)
	if err == nil {
		_ = os.Chtimes(path, parsed, parsed)
	}
	return path
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertManifestHashes(t *testing.T, zipPath string) {
	t.Helper()
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, "manifest.json") {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer handle.Close()
		var manifest reports.BundleManifest
		if err := json.NewDecoder(handle).Decode(&manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Files) == 0 {
			t.Fatal("manifest has no files")
		}
		for _, entry := range manifest.Files {
			if entry.SHA256 == "" {
				t.Fatalf("manifest entry missing sha256: %#v", entry)
			}
		}
		return
	}
	t.Fatal("manifest.json not found in bundle")
}

func samePath(left string, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
