package diagnostics

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kigrepair/internal/app"
)

var safeHistoryFiles = map[string]bool{
	"summary.txt":                true,
	"repair.log":                 true,
	"operations.json":            true,
	"initial-detection.json":     true,
	"final-detection.json":       true,
	"detection.json":             true,
	"repair-result.json":         true,
	"install-result.json":        true,
	"defender-result.json":       true,
	"classification-result.json": true,
	"cleanup-plan.json":          true,
	"collect-result.json":        true,
	"msi-install.log":            true,
	"msi-uninstall.log":          true,
}

type HistoryCollector struct {
	Limit int
}

func (HistoryCollector) Name() string {
	return "history"
}

func (c HistoryCollector) Collect(ctx *app.AppContext) app.OperationResult {
	limit := c.Limit
	if limit <= 0 {
		limit = 5
	}
	dirs, err := selectHistoryDirs(ctx.ReportRoot, ctx.OutputDir, limit)
	if err != nil {
		return operation("collect.history", "history", app.OperationStatusWarning, "History could not be read", err.Error())
	}
	copied, warnings := copyHistoryFiles(dirs, filepath.Join(ctx.OutputDir, "history"))
	if len(warnings) > 0 {
		return operation("collect.history", "history", app.OperationStatusWarning, "Copied history with warnings", strings.Join(warnings, "; "))
	}
	if copied == 0 {
		return operation("collect.history", "history", app.OperationStatusSkipped, "No safe history files found", "")
	}
	return operation("collect.history", "history", app.OperationStatusSuccess, "Copied history", "")
}

func selectHistoryDirs(root string, currentDir string, limit int) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	dirs := make([]string, 0)
	currentClean := filepath.Clean(currentDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if strings.EqualFold(filepath.Clean(path), currentClean) {
			continue
		}
		dirs = append(dirs, path)
	}
	sort.Slice(dirs, func(i, j int) bool {
		left, _ := os.Stat(dirs[i])
		right, _ := os.Stat(dirs[j])
		if left == nil || right == nil {
			return dirs[i] > dirs[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	if limit > 0 && len(dirs) > limit {
		dirs = dirs[:limit]
	}
	return dirs, nil
}

func copyHistoryFiles(dirs []string, targetRoot string) (int, []string) {
	copied := 0
	warnings := make([]string, 0)
	for _, dir := range dirs {
		base := filepath.Base(dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !safeHistoryFiles[entry.Name()] || strings.EqualFold(filepath.Ext(entry.Name()), ".zip") {
				continue
			}
			src := filepath.Join(dir, entry.Name())
			dst := filepath.Join(targetRoot, base, entry.Name())
			if err := copyFile(src, dst); err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			copied++
		}
	}
	return copied, warnings
}

type MSILogsCollector struct {
	Limit int
}

func (MSILogsCollector) Name() string {
	return "msi-logs"
}

func (c MSILogsCollector) Collect(ctx *app.AppContext) app.OperationResult {
	limit := c.Limit
	if limit <= 0 {
		limit = 5
	}
	dirs := []string{ctx.OutputDir}
	historyDirs, err := selectHistoryDirs(ctx.ReportRoot, ctx.OutputDir, limit)
	if err != nil {
		return operation("collect.msi_logs", "msi-logs", app.OperationStatusWarning, "MSI logs could not be searched", err.Error())
	}
	dirs = append(dirs, historyDirs...)
	copied, warnings := copyMSILogs(dirs, filepath.Join(ctx.OutputDir, "msi-logs"))
	if len(warnings) > 0 {
		return operation("collect.msi_logs", "msi-logs", app.OperationStatusWarning, "Copied MSI logs with warnings", strings.Join(warnings, "; "))
	}
	if copied == 0 {
		return operation("collect.msi_logs", "msi-logs", app.OperationStatusSkipped, "No MSI logs found", "")
	}
	return operation("collect.msi_logs", "msi-logs", app.OperationStatusSuccess, "Copied msi-logs", "")
}

func copyMSILogs(dirs []string, targetRoot string) (int, []string) {
	names := map[string]bool{"msi-install.log": true, "msi-uninstall.log": true}
	copied := 0
	warnings := make([]string, 0)
	for _, dir := range dirs {
		base := filepath.Base(dir)
		for name := range names {
			src := filepath.Join(dir, name)
			if _, err := os.Stat(src); err != nil {
				continue
			}
			dst := filepath.Join(targetRoot, base+"-"+name)
			if err := copyFile(src, dst); err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			copied++
		}
	}
	return copied, warnings
}
