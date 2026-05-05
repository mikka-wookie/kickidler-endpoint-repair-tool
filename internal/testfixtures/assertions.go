package testfixtures

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/cleaner"
)

func AssertNoRawInvite(t *testing.T, rootDir string, rawInvite string) {
	t.Helper()
	if strings.TrimSpace(rawInvite) == "" {
		return
	}
	err := filepath.WalkDir(rootDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), rawInvite) {
			t.Fatalf("%s contains raw invite", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func AssertZipNoRawInvite(t *testing.T, zipPath string, rawInvite string) {
	t.Helper()
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), rawInvite) {
			t.Fatalf("%s in %s contains raw invite", file.Name, zipPath)
		}
	}
}

func AssertNoMutations(t *testing.T, recorder *MutationRecorder) {
	t.Helper()
	if recorder != nil && len(recorder.Calls) > 0 {
		t.Fatalf("unexpected mutations: %#v", recorder.Calls)
	}
}

func AssertOnlyPlannedMutations(t *testing.T, plan cleaner.CleanupPlan, recorder *MutationRecorder) {
	t.Helper()
	planned := map[string]int{}
	for _, action := range plan.Actions {
		planned[string(action.Type)+"\x00"+action.Target]++
	}
	for _, call := range recorder.Calls {
		key := call.Type + "\x00" + call.Target
		if planned[key] == 0 {
			t.Fatalf("mutation was not in plan: %#v", call)
		}
		planned[key]--
	}
}

func AssertNoHiddenWindowsProcessKilled(t *testing.T, recorder *MutationRecorder) {
	t.Helper()
	for _, call := range recorder.Calls {
		if call.Type == string(cleaner.CleanupActionKillProcess) && strings.Contains(strings.ToLower(call.Target), `c:\windows\system32\svchost.exe`) {
			t.Fatalf("normal Windows svchost was targeted: %#v", call)
		}
	}
}

func AssertNoPathOutsideRoot(t *testing.T, targets []string, root string) {
	t.Helper()
	root = strings.ToLower(filepath.Clean(root))
	for _, target := range targets {
		cleaned := strings.ToLower(filepath.Clean(target))
		if cleaned != root && !strings.HasPrefix(cleaned, root+string(os.PathSeparator)) {
			t.Fatalf("target outside root: %s outside %s", target, root)
		}
	}
}
