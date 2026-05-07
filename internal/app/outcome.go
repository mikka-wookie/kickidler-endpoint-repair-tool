package app

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"kigrepair/internal/config"
	"kigrepair/internal/safety"
)

const defaultOutcomeTimelineLimit = 25

type OutcomeInput struct {
	Meta     WorkflowResponseMeta
	Result   any
	Warnings []string
	Errors   []string
	Timeline []OperationResult
	Policy   config.PolicySummary
	Err      error
}

func WorkflowStatusForExit(exitCode int, err error, notReady bool) string {
	if errors.Is(err, context.Canceled) {
		return string(WorkflowStatusCancelled)
	}
	if notReady {
		return string(WorkflowStatusNotReady)
	}
	switch exitCode {
	case ExitSuccess:
		return string(WorkflowStatusSuccess)
	case ExitWarnings:
		return string(WorkflowStatusWarning)
	case ExitAdminRequired, ExitConfirmationRequired:
		return string(WorkflowStatusNotReady)
	case ExitUnexpectedError:
		return string(WorkflowStatusFailed)
	default:
		if err != nil {
			return string(WorkflowStatusFailed)
		}
		if exitCode >= ExitInvalidInput {
			return string(WorkflowStatusFailed)
		}
		return string(WorkflowStatusSuccess)
	}
}

func BuildWorkflowOutcome(input OutcomeInput) WorkflowOutcome {
	decoded := decodeMap(input.Result)
	status := WorkflowStatusForExit(input.Meta.ExitCode, input.Err, resultNotReady(decoded, input.Errors))
	files := BuildResultFileSummary(input.Meta.ReportDir, input.Meta.PrimaryResultFile)
	out := WorkflowOutcome{
		RunID:             input.Meta.RunID,
		Workflow:          input.Meta.Workflow,
		Status:            status,
		ExitCode:          input.Meta.ExitCode,
		StartedAt:         input.Meta.StartedAt,
		FinishedAt:        input.Meta.FinishedAt,
		DurationMS:        input.Meta.DurationMS,
		ReportDir:         input.Meta.ReportDir,
		SummaryFile:       input.Meta.SummaryFile,
		OperationsFile:    input.Meta.OperationsFile,
		PrimaryResultFile: input.Meta.PrimaryResultFile,
		Files:             files,
		Policy:            input.Policy,
		Health:            firstString(decoded, "health", "detection_health", "initial_health", "final_health", "status", "classification"),
		InstallMode:       firstString(decoded, "install_mode", "initial_install_mode", "final_install_mode"),
		Admin:             adminSummary(decoded),
		Installer:         installerSummary(decoded),
		Defender:          defenderSummary(decoded),
	}
	if plan := mapField(decoded, "repair_plan"); plan != nil {
		if out.Health == "" {
			out.Health = firstString(plan, "detection_health", "health", "initial_health", "final_health")
		}
		if out.InstallMode == "" {
			out.InstallMode = firstString(plan, "install_mode", "initial_install_mode", "final_install_mode")
		}
		out.RepairReadiness = repairReadiness(plan)
		if out.Admin == (AdminSummary{}) {
			out.Admin = adminSummary(plan)
		}
		if out.Installer == (InstallerSummary{}) {
			out.Installer = installerSummary(plan)
		}
		if out.Defender == (DefenderSummary{}) {
			out.Defender = defenderSummary(plan)
		}
	}
	if out.RepairReadiness == "" {
		out.RepairReadiness = repairReadiness(decoded)
	}
	out.PrimaryIssue = primaryIssueSummary(decoded)
	out.Recommendation = recommendationSummary(decoded)
	out.BlockingReasons = blockingMessages(decoded, input.Errors)
	out.Warnings = aggregateUserMessages(input.Warnings, input.Timeline, files.PreferredDetailsRef(), "warning")
	out.Errors = userMessagesFromErrors(input.Errors)
	out.Timeline = compactTimeline(input.Timeline, files.PreferredDetailsRef(), defaultOutcomeTimelineLimit)
	return out
}

func BuildResultFileSummary(reportDir string, primary string) ResultFileSummary {
	if strings.TrimSpace(reportDir) == "" {
		return ResultFileSummary{}
	}
	file := func(name string) string { return filepath.Join(reportDir, name) }
	return ResultFileSummary{
		ReportDir:            reportDir,
		Summary:              file("summary.txt"),
		Operations:           file("operations.json"),
		InitialDetection:     file("initial-detection.json"),
		FinalDetection:       file("final-detection.json"),
		PreflightResult:      file("preflight-result.json"),
		RepairPlan:           file("repair-plan.json"),
		RepairResult:         file("repair-result.json"),
		CleanupPlan:          file("cleanup-plan.json"),
		CleanupResult:        file("cleanup-result.json"),
		VerificationResult:   file("verification-result.json"),
		ClassificationResult: file("classification-result.json"),
		RecommendationResult: file("recommendation-result.json"),
		InstallerValidation:  file("installer-validation.json"),
		DefenderResult:       file("defender-result.json"),
		SupportBundle:        file("kigrepair-support-bundle.zip"),
		ReportsList:          file("reports-list.json"),
		ReportsCleanupPlan:   file("report-cleanup-plan.json"),
		ReportsCleanupResult: file("report-cleanup-result.json"),
		ConfigShow:           file("config-show.json"),
		ConfigValidation:     file("config-validation.json"),
		PrimaryResult:        primary,
	}
}

func (f ResultFileSummary) PreferredDetailsRef() string {
	for _, value := range []string{f.CleanupPlan, f.Operations, f.PrimaryResult} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ToUserMessage(err error, code string) UserMessage {
	text := ""
	if err != nil {
		text = err.Error()
	}
	return UserMessageFromText(firstNonEmpty(code, text), text)
}

func UserMessageFromText(code string, text string) UserMessage {
	code = safety.RedactString(code)
	text = safety.RedactString(text)
	normalized := strings.ToLower(strings.TrimSpace(code + " " + text))
	msg := UserMessage{Code: normalizeCode(code, normalized), Severity: "error"}
	switch {
	case strings.Contains(normalized, "admin_rights"), strings.Contains(normalized, "administrator"):
		msg.Code = "admin_rights"
		msg.Title = "Administrator rights required"
		msg.Message = "Restart the tool as Administrator to complete service, Defender, and repair checks."
	case strings.Contains(normalized, "file missing"), strings.Contains(normalized, "installer_available"), strings.Contains(normalized, "installer file"), strings.Contains(normalized, "installer not found"), strings.Contains(normalized, "path is required"):
		msg.Code = "installer_missing"
		msg.Title = "Installer file was not found"
		msg.Message = "Select a supported Grabber MSI or place it in the assets folder."
	case strings.Contains(normalized, "invalid installer"), strings.Contains(normalized, "unsupported installer"), strings.Contains(normalized, "failed validation"):
		msg.Code = "invalid_installer"
		msg.Title = "Installer is invalid"
		msg.Message = "The selected installer failed validation. Select a supported Grabber MSI."
	case strings.Contains(normalized, "preflight_not_ready"), strings.Contains(normalized, "not_ready"), strings.Contains(normalized, "not ready"):
		msg.Code = "preflight_not_ready"
		msg.Title = "Repair is not ready"
		msg.Message = "Resolve the blocking preflight checks, then run Preflight again."
	case strings.Contains(normalized, "defender_status_unavailable"), strings.Contains(normalized, "defender status unavailable"), strings.Contains(normalized, "defender exclusions could not be verified"):
		msg.Code = "defender_status_unavailable"
		msg.Title = "Defender status unavailable"
		msg.Message = "Run as Administrator or verify Defender exclusions manually."
	case strings.Contains(normalized, "missing_invite"), strings.Contains(normalized, "invite_present"), strings.Contains(normalized, "invite is required"):
		msg.Code = "missing_invite"
		msg.Title = "Invite is required"
		msg.Message = "Enter the invite value before running this workflow."
	case strings.Contains(normalized, "cancelled"), strings.Contains(normalized, "canceled"):
		msg.Code = "cancelled"
		msg.Title = "Workflow cancelled"
		msg.Message = "The workflow was cancelled before completion."
	default:
		msg.Code = firstNonEmpty(msg.Code, "workflow_failed")
		msg.Title = "Workflow failed"
		msg.Message = "Review the report directory or collect a support bundle."
	}
	return msg
}

func compactTimeline(results []OperationResult, detailsRef string, limit int) []TimelineItem {
	counts := warningCounts(nil, results)
	items := make([]TimelineItem, 0, len(results)+len(counts)+1)
	for _, result := range results {
		if warningAggregationKey(result.Message) != "" {
			continue
		}
		items = append(items, TimelineItem{
			OperationID:     firstNonEmpty(result.ID, result.Step),
			Status:          string(result.Status),
			Message:         safety.RedactString(result.Message),
			DurationMS:      result.DurationMS,
			FailureCategory: result.FailureCategory,
			DetailsRef:      firstNonEmpty(result.ResultFile, result.Artifact),
		})
	}
	for _, key := range sortedCountKeys(counts) {
		items = append(items, TimelineItem{
			OperationID: "warning-summary",
			Status:      string(OperationStatusWarning),
			Message:     key + ": " + itoa(counts[key]) + ".",
			DetailsRef:  detailsRef,
		})
	}
	if limit <= 0 || len(items) <= limit {
		return items
	}
	capped := append([]TimelineItem{}, items[:limit]...)
	capped = append(capped, TimelineItem{
		OperationID: "details",
		Status:      string(OperationStatusWarning),
		Message:     "Additional details saved in operations.json.",
		DetailsRef:  detailsRef,
	})
	return capped
}

func aggregateUserMessages(warnings []string, results []OperationResult, detailsRef string, severity string) []UserMessage {
	counts := warningCounts(warnings, results)
	messages := make([]UserMessage, 0, len(counts)+len(warnings))
	for _, key := range sortedCountKeys(counts) {
		messages = append(messages, UserMessage{
			Code:       warningCode(key),
			Severity:   severity,
			Title:      key,
			Message:    key + ": " + itoa(counts[key]) + ".",
			DetailsRef: detailsRef,
		})
	}
	for _, warning := range warnings {
		if warningAggregationKey(warning) != "" {
			continue
		}
		messages = append(messages, UserMessage{Severity: severity, Title: "Workflow warning", Message: safety.RedactString(warning), DetailsRef: detailsRef})
	}
	return dedupeUserMessages(messages)
}

func warningCounts(warnings []string, results []OperationResult) map[string]int {
	counts := map[string]int{}
	for _, warning := range warnings {
		if key := warningAggregationKey(warning); key != "" {
			counts[key]++
		}
	}
	for _, result := range results {
		if key := warningAggregationKey(result.Message); key != "" {
			counts[key]++
		}
	}
	return counts
}

func warningAggregationKey(message string) string {
	normalized := strings.ToLower(message)
	switch {
	case strings.Contains(normalized, "skipped unsafe process match"):
		return "Skipped unsafe process matches"
	case strings.Contains(normalized, "hidden wmi process name but executable path is unavailable"):
		return "Hidden WMI process names with unavailable paths"
	case strings.Contains(normalized, "normal windows process path, not grabber hidden wmi path"):
		return "Normal Windows process path mismatches skipped"
	case strings.Contains(normalized, "known grabber process name but executable path is unavailable"):
		return "Known Grabber process names with unavailable paths skipped"
	default:
		return ""
	}
}

func blockingMessages(decoded map[string]any, errors []string) []UserMessage {
	var out []UserMessage
	for _, source := range []map[string]any{decoded, mapField(decoded, "repair_plan"), mapField(decoded, "preflight")} {
		if source == nil {
			continue
		}
		if checks, ok := source["checks"].([]any); ok {
			for _, raw := range checks {
				check, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				required, _ := boolField(check, "required")
				status := strings.ToLower(firstString(check, "status"))
				if required && (status == "fail" || status == "failed") {
					out = append(out, UserMessageFromText(firstString(check, "code", "name"), firstString(check, "error", "message", "action")))
				}
			}
		}
		for _, value := range append(stringSlice(source, "failed_required_checks"), stringSlice(source, "required_inputs")...) {
			out = append(out, UserMessageFromText(value, value))
		}
	}
	for _, errText := range errors {
		msg := UserMessageFromText(errText, errText)
		if msg.Code != "workflow_failed" {
			out = append(out, msg)
		}
	}
	return dedupeUserMessages(out)
}

func userMessagesFromErrors(errors []string) []UserMessage {
	out := make([]UserMessage, 0, len(errors))
	for _, errText := range errors {
		out = append(out, UserMessageFromText(errText, errText))
	}
	return dedupeUserMessages(out)
}

func adminSummary(decoded map[string]any) AdminSummary {
	for _, source := range []map[string]any{decoded, mapField(decoded, "environment"), mapField(decoded, "preflight"), mapField(mapField(decoded, "repair_plan"), "preflight")} {
		if source == nil {
			continue
		}
		if isAdmin, ok := boolField(source, "is_admin", "admin_rights"); ok {
			msg := ""
			if !isAdmin {
				msg = "Administrator rights required"
			}
			return AdminSummary{IsAdmin: isAdmin, LimitedMode: !isAdmin, Message: msg}
		}
	}
	return AdminSummary{}
}

func installerSummary(decoded map[string]any) InstallerSummary {
	for _, source := range []map[string]any{mapField(decoded, "installer"), mapField(mapField(decoded, "repair_plan"), "installer"), mapField(mapField(decoded, "repair_plan"), "preflight")} {
		if source == nil {
			continue
		}
		exists, existsOK := boolField(source, "exists", "installer_found", "installer_available", "discovered")
		readable, readableOK := boolField(source, "readable", "installer_readable")
		path := firstString(source, "path", "installer_path")
		valid := exists
		if validation := mapField(source, "validation"); validation != nil {
			if v, ok := boolField(validation, "valid", "supported_name", "readable"); ok {
				valid = exists && v
			}
		}
		if existsOK || readableOK || path != "" {
			msg := ""
			status := "available"
			if !exists {
				status = "missing"
				msg = "Installer file was not found"
			} else if !valid {
				status = "invalid"
				msg = "Installer is invalid"
			}
			return InstallerSummary{Path: path, Exists: exists, Readable: readable, Valid: valid, Status: status, Message: msg}
		}
	}
	return InstallerSummary{}
}

func defenderSummary(decoded map[string]any) DefenderSummary {
	for _, source := range []map[string]any{mapField(decoded, "defender"), mapField(mapField(decoded, "repair_plan"), "defender")} {
		if source == nil {
			continue
		}
		required, _ := boolField(source, "required")
		covered, _ := boolField(source, "covered", "already_covered")
		status := firstString(source, "status", "read_status")
		msg := ""
		if status == "unavailable" {
			msg = "Defender status unavailable"
		}
		return DefenderSummary{Status: status, Required: required, Covered: covered, Message: msg}
	}
	return DefenderSummary{}
}

func primaryIssueSummary(decoded map[string]any) *IssueSummary {
	for _, source := range []map[string]any{decoded, mapField(decoded, "repair_plan")} {
		classification := mapField(source, "classification")
		issue := mapField(classification, "primary_issue")
		if issue == nil {
			continue
		}
		code := firstString(issue, "code")
		if code == "" {
			continue
		}
		return &IssueSummary{
			Code:     code,
			Severity: firstNonEmpty(firstString(issue, "severity"), "warning"),
			Title:    firstNonEmpty(firstString(issue, "title"), issueTitle(code)),
			Message:  firstString(issue, "message", "description"),
			Action:   firstString(issue, "action"),
		}
	}
	return nil
}

func recommendationSummary(decoded map[string]any) *ActionSummary {
	for _, source := range []map[string]any{decoded, mapField(decoded, "repair_plan")} {
		recommendation := mapField(source, "recommendation")
		action := mapField(recommendation, "primary_action")
		if action == nil {
			continue
		}
		code := firstString(action, "code")
		if code == "" {
			continue
		}
		destructive, _ := boolField(action, "destructive")
		requiresAdmin, _ := boolField(action, "requires_admin")
		requiresYes, _ := boolField(action, "requires_yes")
		return &ActionSummary{
			Code:          code,
			Title:         firstNonEmpty(firstString(action, "title", "message"), recommendationTitle(code)),
			Description:   firstString(action, "description"),
			Command:       sanitizeOutcomeCommand(firstString(action, "command")),
			Destructive:   destructive,
			RequiresAdmin: requiresAdmin,
			RequiresYes:   requiresYes,
		}
	}
	return nil
}

func resultNotReady(decoded map[string]any, errors []string) bool {
	for _, source := range []map[string]any{decoded, mapField(decoded, "repair_plan")} {
		if source == nil {
			continue
		}
		if strings.EqualFold(firstString(source, "status"), "not_ready") {
			return true
		}
		if ready, ok := boolField(source, "ready_for_repair"); ok && !ready {
			return true
		}
	}
	for _, errText := range errors {
		msg := UserMessageFromText(errText, errText)
		if msg.Code == "admin_rights" || msg.Code == "installer_missing" || msg.Code == "invalid_installer" || msg.Code == "missing_invite" || msg.Code == "preflight_not_ready" {
			return true
		}
	}
	return false
}

func repairReadiness(decoded map[string]any) string {
	if decoded == nil {
		return ""
	}
	if strings.EqualFold(firstString(decoded, "status"), "not_ready") {
		return "not_ready"
	}
	if ready, ok := boolField(decoded, "ready_for_repair"); ok {
		if ready {
			return "ready"
		}
		return "not_ready"
	}
	return ""
}

func decodeMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil
	}
	return decoded
}

func mapField(decoded map[string]any, key string) map[string]any {
	if decoded == nil {
		return nil
	}
	for _, candidate := range keyCandidates(key) {
		if value, ok := decoded[candidate].(map[string]any); ok {
			return value
		}
	}
	return nil
}

func firstString(decoded map[string]any, keys ...string) string {
	for _, key := range keys {
		for _, candidate := range keyCandidates(key) {
			if value, ok := decoded[candidate].(string); ok {
				return safety.RedactString(value)
			}
		}
	}
	return ""
}

func boolField(decoded map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		for _, candidate := range keyCandidates(key) {
			if value, ok := decoded[candidate].(bool); ok {
				return value, true
			}
		}
	}
	return false, false
}

func stringSlice(decoded map[string]any, keys ...string) []string {
	for _, key := range keys {
		for _, candidate := range keyCandidates(key) {
			raw, ok := decoded[candidate].([]any)
			if !ok {
				continue
			}
			out := make([]string, 0, len(raw))
			for _, value := range raw {
				if s, ok := value.(string); ok {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func keyCandidates(key string) []string {
	if key == "" {
		return nil
	}
	return []string{key, strings.ToLower(key[:1]) + key[1:], camelToSnake(key)}
}

func camelToSnake(value string) string {
	var b strings.Builder
	for i, r := range value {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func sortedCountKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func dedupeUserMessages(values []UserMessage) []UserMessage {
	out := make([]UserMessage, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		key := value.Code + "\x00" + value.Title + "\x00" + value.Message
		if value.Title == "" || value.Message == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func normalizeCode(code string, normalized string) string {
	code = strings.TrimSpace(strings.ToLower(code))
	code = strings.ReplaceAll(code, " ", "_")
	code = strings.Trim(code, ":")
	if code != "" && !strings.Contains(code, ":") {
		return code
	}
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return ""
	}
	return strings.Trim(parts[0], ":")
}

func issueTitle(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "partial_msi_leftovers":
		return "MSI registry leftovers found"
	case "service_binary_missing":
		return "Service executable is missing"
	case "service_not_running", "service_stopped":
		return "Grabber service is stopped"
	case "defender_status_unavailable":
		return "Defender status unavailable"
	default:
		return code
	}
}

func recommendationTitle(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "run_cleanup_dry_run", "cleanup_dry_run":
		return "Run cleanup dry-run"
	case "collect_bundle", "collect_report", "collect_support_bundle":
		return "Collect support bundle"
	case "run_repair", "run_full_repair":
		return "Run repair"
	default:
		return code
	}
}

func sanitizeOutcomeCommand(command string) string {
	command = safety.RedactString(command)
	command = strings.ReplaceAll(command, "invite=<REDACTED>", "invite=<INVITE>")
	lower := strings.ToLower(command)
	idx := strings.Index(lower, "--invite")
	if idx < 0 {
		return command
	}
	restStart := idx + len("--invite")
	prefix := command[:restStart]
	rest := strings.TrimLeft(command[restStart:], " \t=")
	if rest == "" {
		return command
	}
	if rest[0] == '"' {
		if end := strings.Index(rest[1:], `"`); end >= 0 {
			return prefix + ` "<INVITE>"` + rest[end+2:]
		}
		return prefix + ` "<INVITE>"`
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return command
	}
	return prefix + " <INVITE>" + strings.TrimPrefix(rest[len(fields[0]):], " ")
}

func warningCode(title string) string {
	return strings.ReplaceAll(strings.ToLower(title), " ", "_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	n := value
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
