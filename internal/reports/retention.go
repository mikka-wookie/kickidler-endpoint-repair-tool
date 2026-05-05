package reports

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"kigrepair/internal/safety"
)

const (
	DefaultRetentionOlderThan = "30d"
	DefaultRetentionKeepLast  = 10
)

var knownReportFiles = map[string]bool{
	"initial-detection.json":       true,
	"final-detection.json":         true,
	"operations.json":              true,
	"summary.txt":                  true,
	"repair.log":                   true,
	"cleanup-plan.json":            true,
	"cleanup-result.json":          true,
	"install-result.json":          true,
	"defender-result.json":         true,
	"repair-result.json":           true,
	"collect-result.json":          true,
	"verification-result.json":     true,
	"classification-result.json":   true,
	"recommendation-result.json":   true,
	"preflight-result.json":        true,
	"repair-plan.json":             true,
	"installer-validation.json":    true,
	"rollback-info.json":           true,
	"report-cleanup-plan.json":     true,
	"report-cleanup-result.json":   true,
	"msi-install.log":              true,
	"msi-uninstall.log":            true,
	"kigrepair-support-bundle.zip": true,
}

func ListReports(root string, opts ReportListOptions) (ReportListResult, error) {
	root = cleanRoot(root)
	result := ReportListResult{Command: "reports list", ReportsRoot: root, Reports: []ReportEntry{}}
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		report, err := inspectReportDir(root, entry.Name())
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		result.TotalSizeBytes += report.SizeBytes
		result.Reports = append(result.Reports, report)
	}
	sortReportsNewestFirst(result.Reports)
	if opts.Limit > 0 && !opts.All && len(result.Reports) > opts.Limit {
		result.Reports = result.Reports[:opts.Limit]
	}
	result.Count = len(result.Reports)
	return result, nil
}

func BuildReportCleanupPlan(root string, opts ReportCleanupOptions) (ReportCleanupPlan, error) {
	root = cleanRoot(root)
	olderThanText := strings.TrimSpace(opts.OlderThan)
	if olderThanText == "" {
		olderThanText = DefaultRetentionOlderThan
	}
	keepLast := opts.KeepLast
	if keepLast < 0 {
		keepLast = DefaultRetentionKeepLast
	}
	duration, err := ParseRetentionDuration(olderThanText)
	now := time.Now()
	if strings.TrimSpace(opts.Now) != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, opts.Now); parseErr == nil {
			now = parsed
		}
	}
	plan := ReportCleanupPlan{
		SchemaVersion:  1,
		CreatedAt:      now.Format(time.RFC3339),
		ReportsRoot:    root,
		ReportDir:      opts.ActiveReportDir,
		DryRun:         opts.DryRun,
		OlderThan:      olderThanText,
		KeepLast:       keepLast,
		Candidates:     []ReportEntry{},
		PlannedDeletes: []ReportDeleteTarget{},
		Skipped:        []ReportSkip{},
	}
	if err != nil {
		plan.Errors = append(plan.Errors, err.Error())
		return plan, err
	}

	list, listErr := ListReports(root, ReportListOptions{All: true})
	if listErr != nil {
		plan.Errors = append(plan.Errors, listErr.Error())
		return plan, listErr
	}
	plan.Candidates = list.Reports
	plan.TotalSizeBytes = list.TotalSizeBytes
	plan.Warnings = append(plan.Warnings, list.Warnings...)

	cutoff := now.Add(-duration)
	for index, report := range plan.Candidates {
		if sameCleanPath(report.Path, opts.ActiveReportDir) {
			plan.Skipped = append(plan.Skipped, ReportSkip{Name: report.Name, Path: report.Path, Reason: "active report directory"})
			continue
		}
		if index < keepLast {
			plan.Skipped = append(plan.Skipped, ReportSkip{Name: report.Name, Path: report.Path, Reason: fmt.Sprintf("kept by keep-last=%d", keepLast)})
			continue
		}
		modifiedAt, ok := reportSortTime(report)
		if !ok {
			plan.Skipped = append(plan.Skipped, ReportSkip{Name: report.Name, Path: report.Path, Reason: "timestamp unavailable"})
			plan.Warnings = append(plan.Warnings, report.Name+": timestamp unavailable")
			continue
		}
		if !modifiedAt.Before(cutoff) {
			plan.Skipped = append(plan.Skipped, ReportSkip{Name: report.Name, Path: report.Path, Reason: "not older than retention threshold"})
			continue
		}
		if !hasKnownReportFile(report.Files) {
			plan.Skipped = append(plan.Skipped, ReportSkip{Name: report.Name, Path: report.Path, Reason: "does not match known report directory shape"})
			plan.Warnings = append(plan.Warnings, report.Name+": does not contain known report files")
			continue
		}
		if err := ValidateReportDeleteTarget(root, report.Path, opts.ActiveReportDir); err != nil {
			plan.Skipped = append(plan.Skipped, ReportSkip{Name: report.Name, Path: report.Path, Reason: err.Error()})
			plan.Warnings = append(plan.Warnings, report.Name+": "+err.Error())
			continue
		}
		target := ReportDeleteTarget{
			ID:         strconv.Itoa(len(plan.PlannedDeletes) + 1),
			Name:       report.Name,
			Path:       filepath.Clean(report.Path),
			SizeBytes:  report.SizeBytes,
			ModifiedAt: report.ModifiedAt,
			Reason:     fmt.Sprintf("older than %s after keeping newest %d reports", olderThanText, keepLast),
			Validated:  true,
		}
		plan.PlannedDeletes = append(plan.PlannedDeletes, target)
		plan.PlannedFreeBytes += target.SizeBytes
	}
	return plan, nil
}

func ExecuteReportCleanupPlan(root string, plan ReportCleanupPlan, yes bool) ReportCleanupResult {
	result := ReportCleanupResult{
		Command:          "reports cleanup",
		Status:           "success",
		ReportsRoot:      cleanRoot(root),
		ReportDir:        plan.ReportDir,
		DryRun:           plan.DryRun,
		PlanPath:         filepath.Join(plan.ReportDir, "report-cleanup-plan.json"),
		PlannedFreeBytes: plan.PlannedFreeBytes,
		Skipped:          append([]ReportSkip{}, plan.Skipped...),
	}
	if plan.DryRun {
		result.ExitCode = 0
		return result
	}
	if !yes {
		result.Status = "failed"
		result.ExitCode = 7
		result.Errors = append(result.Errors, "real report cleanup requires --yes")
		return result
	}
	for _, target := range plan.PlannedDeletes {
		outcome := ReportDeleteOutcome{
			ID:        target.ID,
			Name:      target.Name,
			Path:      target.Path,
			SizeBytes: target.SizeBytes,
		}
		if !target.Validated {
			outcome.Status = "failed"
			outcome.Errors = append(outcome.Errors, "target was not validated in cleanup plan")
			result.Deleted = append(result.Deleted, outcome)
			continue
		}
		if err := ValidateReportDeleteTarget(result.ReportsRoot, target.Path, plan.ReportDir); err != nil {
			outcome.Status = "failed"
			outcome.Errors = append(outcome.Errors, err.Error())
			result.Deleted = append(result.Deleted, outcome)
			continue
		}
		if err := os.RemoveAll(target.Path); err != nil {
			outcome.Status = "failed"
			outcome.Errors = append(outcome.Errors, err.Error())
			result.Deleted = append(result.Deleted, outcome)
			continue
		}
		outcome.Status = "success"
		outcome.Message = "report directory deleted"
		result.FreedBytes += target.SizeBytes
		result.Deleted = append(result.Deleted, outcome)
	}
	result.Status = cleanupResultStatus(result)
	switch result.Status {
	case "success":
		result.ExitCode = 0
	case "warning":
		result.ExitCode = 1
	default:
		result.ExitCode = 7
	}
	return result
}

func ValidateReportDeleteTarget(root string, target string, activeReportDir string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(strings.TrimSpace(target))
	if target == "" {
		return errors.New("empty target rejected")
	}
	if strings.Contains(filepath.ToSlash(target), "../") || strings.Contains(filepath.ToSlash(target), "/..") {
		return errors.New("path traversal rejected")
	}
	if !filepath.IsAbs(target) {
		return errors.New("relative target rejected")
	}
	if sameCleanPath(target, root) {
		return errors.New("reports root rejected")
	}
	if sameCleanPath(target, filepath.Dir(root)) || sameCleanPath(target, filepath.Dir(filepath.Dir(root))) {
		return errors.New("parent report path rejected")
	}
	parent := filepath.Dir(target)
	if !sameCleanPath(parent, root) {
		return errors.New("target is not a direct child of reports root")
	}
	if !strings.HasPrefix(strings.ToLower(target), strings.ToLower(root+string(os.PathSeparator))) {
		return errors.New("target outside reports root rejected")
	}
	if activeReportDir != "" && sameCleanPath(target, activeReportDir) {
		return errors.New("active report directory rejected")
	}
	info, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("target is not a directory")
	}
	if isReparsePoint(info) {
		return errors.New("reparse point or symlink rejected")
	}
	files, err := reportFileNames(target)
	if err != nil {
		return err
	}
	if !hasKnownReportFile(files) {
		return errors.New("target does not contain known report files")
	}
	return nil
}

func ParseRetentionDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, errors.New("empty duration")
	}
	if strings.HasSuffix(value, "d") {
		number := strings.TrimSuffix(value, "d")
		days, err := strconv.Atoi(number)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid day duration %q", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(value)
}

func inspectReportDir(root string, name string) (ReportEntry, error) {
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if err != nil {
		return ReportEntry{}, err
	}
	entry := ReportEntry{
		Name:       name,
		Path:       filepath.Clean(path),
		ModifiedAt: info.ModTime().Format(time.RFC3339),
		Files:      []string{},
	}
	if parsed, ok := parseReportTime(name); ok {
		entry.CreatedAt = parsed.Format(time.RFC3339)
	}
	if isReparsePoint(info) {
		entry.Warnings = append(entry.Warnings, "reparse point or symlink")
		return entry, nil
	}
	files, err := reportFileNames(path)
	if err != nil {
		entry.Warnings = append(entry.Warnings, err.Error())
		return entry, nil
	}
	entry.Files = files
	entry.HasSupportBundle = containsFile(files, SupportBundleName)
	entry.Workflow = inferWorkflow(files)
	entry.Status = inferStatus(path, files)
	size, err := dirSize(path)
	if err != nil {
		entry.Warnings = append(entry.Warnings, err.Error())
	}
	entry.SizeBytes = size
	return entry, nil
}

func reportFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}

func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if isReparsePoint(info) && path != root {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func inferWorkflow(files []string) string {
	switch {
	case containsFile(files, "report-cleanup-result.json"):
		return "reports cleanup"
	case containsFile(files, "collect-result.json"):
		return "collect-report"
	case containsFile(files, "repair-result.json"):
		return "repair"
	case containsFile(files, "verification-result.json"):
		return "verify"
	case containsFile(files, "preflight-result.json"):
		return "preflight"
	case containsFile(files, "defender-result.json"):
		return "defender"
	case containsFile(files, "install-result.json"):
		return "install"
	case containsFile(files, "cleanup-result.json"), containsFile(files, "cleanup-plan.json"):
		return "cleanup"
	case containsFile(files, "initial-detection.json"):
		return "check"
	default:
		return ""
	}
}

func inferStatus(dir string, files []string) string {
	for _, name := range []string{
		"report-cleanup-result.json",
		"collect-result.json",
		"repair-result.json",
		"verification-result.json",
		"preflight-result.json",
		"defender-result.json",
		"install-result.json",
		"cleanup-result.json",
	} {
		if containsFile(files, name) {
			if status := statusFromJSON(filepath.Join(dir, name)); status != "" {
				return status
			}
		}
	}
	if containsFile(files, "operations.json") {
		if status := statusFromOperations(filepath.Join(dir, "operations.json")); status != "" {
			return status
		}
	}
	if containsFile(files, "summary.txt") {
		return statusFromSummary(filepath.Join(dir, "summary.txt"))
	}
	return ""
}

func statusFromJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return ""
	}
	if status, ok := value["status"].(string); ok {
		return strings.TrimSpace(status)
	}
	if status, ok := value["overall_status"].(string); ok {
		return strings.TrimSpace(status)
	}
	return ""
}

func statusFromOperations(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var ops []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &ops); err != nil {
		return ""
	}
	status := "success"
	for _, op := range ops {
		switch op.Status {
		case "failed":
			return "failed"
		case "warning":
			status = "warning"
		}
	}
	return status
}

func statusFromSummary(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(safety.RedactString(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "status:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
		}
	}
	return ""
}

func parseReportTime(name string) (time.Time, bool) {
	for _, layout := range []string{TimestampLayout, "20060102-150405", "2006-01-02-15-04-05"} {
		if parsed, err := time.ParseInLocation(layout, name, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func sortReportsNewestFirst(entries []ReportEntry) {
	sort.Slice(entries, func(i, j int) bool {
		left, leftOK := reportSortTime(entries[i])
		right, rightOK := reportSortTime(entries[j])
		if leftOK && rightOK {
			return left.After(right)
		}
		if leftOK != rightOK {
			return leftOK
		}
		return entries[i].Name > entries[j].Name
	})
}

func reportSortTime(entry ReportEntry) (time.Time, bool) {
	if parsed, ok := parseReportTime(entry.Name); ok {
		return parsed, true
	}
	if entry.ModifiedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, entry.ModifiedAt); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func cleanupResultStatus(result ReportCleanupResult) string {
	hasFailure := false
	hasWarning := len(result.Warnings) > 0
	for _, deleted := range result.Deleted {
		switch deleted.Status {
		case "failed":
			hasFailure = true
		case "warning", "skipped":
			hasWarning = true
		}
	}
	if hasFailure {
		return "warning"
	}
	if hasWarning {
		return "warning"
	}
	return "success"
}

func cleanRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(`C:\ProgramData`, "kigrepair", "Reports")
	}
	return filepath.Clean(root)
}

func hasKnownReportFile(files []string) bool {
	for _, file := range files {
		if knownReportFiles[file] {
			return true
		}
	}
	return false
}

func containsFile(files []string, name string) bool {
	for _, file := range files {
		if strings.EqualFold(file, name) {
			return true
		}
	}
	return false
}

func sameCleanPath(left string, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
