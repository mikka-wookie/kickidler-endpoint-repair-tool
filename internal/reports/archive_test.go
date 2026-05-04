package reports

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSupportBundleExcludesItselfAndPreservesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "eventlogs")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "README.txt"), []byte("events"), 0644); err != nil {
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
	for _, file := range reader.File {
		names[file.Name] = true
	}
	if names[SupportBundleName] {
		t.Fatalf("bundle included itself")
	}
	if !names["summary.txt"] {
		t.Fatalf("summary.txt was not included")
	}
	if !names["eventlogs/README.txt"] {
		t.Fatalf("nested eventlogs/README.txt was not included")
	}
}
