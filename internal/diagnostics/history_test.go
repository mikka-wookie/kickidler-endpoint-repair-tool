package diagnostics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectHistoryDirsSkipsCurrentAndLimitsLatest(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "2026-05-04_10-00-00")
	oldest := filepath.Join(root, "2026-05-01_10-00-00")
	middle := filepath.Join(root, "2026-05-02_10-00-00")
	latest := filepath.Join(root, "2026-05-03_10-00-00")
	for _, dir := range []string{current, oldest, middle, latest} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	_ = os.Chtimes(oldest, now.Add(-3*time.Hour), now.Add(-3*time.Hour))
	_ = os.Chtimes(middle, now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	_ = os.Chtimes(latest, now.Add(-1*time.Hour), now.Add(-1*time.Hour))
	_ = os.Chtimes(current, now, now)

	dirs, err := selectHistoryDirs(root, current, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 2 {
		t.Fatalf("len(dirs) = %d", len(dirs))
	}
	if dirs[0] != latest || dirs[1] != middle {
		t.Fatalf("dirs = %#v", dirs)
	}
}

func TestCopyHistoryFilesUsesAllowlistAndSkipsZip(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"summary.txt":                  "safe",
		"repair.log":                   "safe",
		"kigrepair-support-bundle.zip": "skip",
		"unknown.txt":                  "skip",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(source, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(root, "target")
	copied, warnings := copyHistoryFiles([]string{source}, target)
	if len(warnings) > 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if copied != 2 {
		t.Fatalf("copied = %d", copied)
	}
	if _, err := os.Stat(filepath.Join(target, "source", "summary.txt")); err != nil {
		t.Fatalf("summary.txt not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "source", "repair.log")); err != nil {
		t.Fatalf("repair.log not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "source", "unknown.txt")); !os.IsNotExist(err) {
		t.Fatalf("unknown.txt should not be copied")
	}
	if _, err := os.Stat(filepath.Join(target, "source", "kigrepair-support-bundle.zip")); !os.IsNotExist(err) {
		t.Fatalf("zip should not be copied")
	}
}
