package ui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"kigrepair/internal/app"
	"kigrepair/internal/safety"
)

func BuildResultView(resp *app.WorkflowResponse) ResultView {
	if resp == nil {
		return ResultView{Status: string(app.WorkflowStatusFailed), Errors: []string{"workflow returned no response"}}
	}
	state := MapWorkflowResponseToGUIState(resp)
	view := resultViewFromGUIState(state)
	view.Errors = redactSlice(resp.Errors)
	view.Reports = append([]string(nil), state.Files.ReportDir)
	for _, op := range resp.Timeline {
		view.Timeline = append(view.Timeline, TimelineItem{
			Time:            op.Timestamp,
			Stage:           "operation",
			Step:            safety.RedactString(firstNonEmptyString(op.ID, op.Step)),
			Target:          safety.RedactString(op.Target),
			Status:          string(op.Status),
			Message:         safety.RedactString(op.Message),
			DurationMS:      op.DurationMS,
			FailureCategory: safety.RedactString(op.FailureCategory),
			DetailsFile:     safety.RedactString(firstNonEmptyString(op.ResultFile, op.Artifact)),
		})
	}
	view.Timeline = AggregateTimelineWarnings(view.Timeline, view.Warnings, detailsFileForAggregation(view))
	if view.SupportBundlePath == "" && view.ReportDir != "" && resp.Meta.Workflow == "collect-report" {
		view.SupportBundlePath = filepath.Join(view.ReportDir, "kigrepair-support-bundle.zip")
	}
	return view
}

func MapWorkflowResponseToGUIState(resp *app.WorkflowResponse) GUIState {
	if resp == nil {
		return GUIState{CurrentStatus: "Failed", BlockingReasons: []string{"Workflow returned no response."}}
	}
	state := GUIState{
		CurrentWorkflow: resp.Meta.Workflow,
		CurrentStatus:   supportStatus(resp.Meta.Status, resp.Result, resp.Errors),
		LatestRunID:     resp.Meta.RunID,
		LatestReportDir: resp.Meta.ReportDir,
		Warnings:        redactSlice(resp.Warnings),
		Files: ResultFiles{
			ReportDir:         resp.Meta.ReportDir,
			SummaryFile:       resp.Meta.SummaryFile,
			OperationsFile:    resp.Meta.OperationsFile,
			PrimaryResultFile: resp.Meta.PrimaryResultFile,
		},
	}
	extractStateSummary(&state, resp.Result)
	state.BlockingReasons = append(state.BlockingReasons, blockingReasonsFromErrors(resp.Errors)...)
	state.BlockingReasons = dedupeStrings(state.BlockingReasons)
	state.Warnings = dedupeStrings(state.Warnings)
	if state.Files.CleanupPlanFile == "" && state.Files.ReportDir != "" {
		state.Files.CleanupPlanFile = filepath.Join(state.Files.ReportDir, "cleanup-plan.json")
	}
	if state.Files.SupportBundlePath == "" && state.Files.ReportDir != "" && resp.Meta.Workflow == "collect-report" {
		state.Files.SupportBundlePath = filepath.Join(state.Files.ReportDir, "kigrepair-support-bundle.zip")
	}
	return state
}

func FormatTimeline(items []TimelineItem) string {
	if len(items) == 0 {
		return "No workflow has been run yet.\r\nRecommended start: Check."
	}
	var b strings.Builder
	for _, item := range items {
		status := supportIcon(item.Status)
		label := item.Step
		if label == "" {
			label = item.Stage
		}
		if label == "" {
			label = "workflow"
		}
		fmt.Fprintf(&b, "%s %s %s", status, label, supportMessage(item.Message))
		if item.Target != "" {
			fmt.Fprintf(&b, " (%s)", item.Target)
		}
		if item.DurationMS > 0 {
			fmt.Fprintf(&b, " in %d ms", item.DurationMS)
		}
		if item.FailureCategory != "" && strings.EqualFold(item.Status, string(app.OperationStatusFailed)) {
			fmt.Fprintf(&b, " - %s", item.FailureCategory)
		}
		if item.DetailsFile != "" {
			fmt.Fprintf(&b, " Details: %s", item.DetailsFile)
		}
		b.WriteString("\r\n")
	}
	return strings.TrimRight(b.String(), "\r\n")
}

func FormatResultSummary(view ResultView) string {
	lines := []string{
		"Status: " + emptyAs(view.Status, "unknown"),
		"Workflow: " + emptyAs(view.Workflow, "unknown"),
		"Run ID: " + emptyAs(view.RunID, "not available"),
		"Report: " + emptyAs(view.ReportDir, "not available"),
	}
	if view.Classification != "" {
		lines = append(lines, "Health: "+view.Classification)
	}
	if view.InstallMode != "" {
		lines = append(lines, "Install mode: "+view.InstallMode)
	}
	if view.PrimaryIssueCode != "" {
		lines = append(lines, "Primary issue: "+view.PrimaryIssueCode+" "+view.PrimaryIssueTitle)
	}
	if view.RecommendationTitle != "" {
		lines = append(lines, "Recommendation: "+view.RecommendationTitle)
	} else if view.Recommendation != "" {
		lines = append(lines, "Recommendation: "+view.Recommendation)
	} else if view.NextRecommendedAction != "" {
		lines = append(lines, "Recommendation: "+view.NextRecommendedAction)
	}
	if len(view.BlockingReasons) > 0 {
		lines = append(lines, "Repair readiness: Not ready")
		lines = append(lines, "Blocking reasons: "+strings.Join(view.BlockingReasons, "; "))
	}
	if len(view.Errors) > 0 {
		lines = append(lines, "Errors: "+strings.Join(view.Errors, "; "))
	}
	if len(view.Warnings) > 0 {
		lines = append(lines, "Warnings: "+strings.Join(view.Warnings, "; "))
	}
	return safety.RedactString(strings.Join(lines, "\r\n"))
}

func CatalogActionNames(catalog app.WorkflowCatalog) map[Action]app.WorkflowCatalogEntry {
	result := map[Action]app.WorkflowCatalogEntry{}
	for _, entry := range catalog.Workflows {
		switch entry.Name {
		case string(ActionCheck), string(ActionVerify), string(ActionPreflight), string(ActionRepair), string(ActionCollectReport), string(ActionReportsList):
			result[Action(entry.Name)] = entry
		case string(ActionRepairDryRun):
			result[ActionRepairDryRun] = entry
		case string(ActionReportsCleanupDry):
			result[ActionReportsCleanupDry] = entry
		}
	}
	return result
}

func extractResultSummary(view *ResultView, value any) {
	if value == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return
	}
	view.Classification = firstString(decoded, "Health", "Status", "Classification")
	view.Recommendation = firstString(decoded, "Recommendation", "Summary")
	view.PrimaryIssueCode = firstString(decoded, "PrimaryIssueCode", "PrimaryIssue", "IssueCode")
	view.NextRecommendedAction = firstString(decoded, "NextRecommendedAction", "NextAction")
	view.SupportBundlePath = firstString(decoded, "BundlePath", "SupportBundlePath")
	view.InstallMode = firstString(decoded, "InstallMode", "InitialInstallMode", "FinalInstallMode")
	extractNestedResultSummary(view, decoded)
	if reports := stringSlice(decoded, "Reports", "ReportDirs"); len(reports) > 0 {
		view.Reports = reports
	}
}

func FormatSystemStatus(view ResultView) string {
	if strings.TrimSpace(view.Workflow) == "" && strings.TrimSpace(view.RunID) == "" {
		return "No workflow has been run yet. Start with Check or Verify."
	}
	lines := []string{
		"Health: " + emptyAs(view.Classification, "unknown") + "       Install mode: " + emptyAs(view.InstallMode, "unknown"),
		"Primary issue: " + emptyAs(strings.TrimSpace(view.PrimaryIssueCode+" "+view.PrimaryIssueTitle), "not available"),
		"Recommendation: " + emptyAs(firstNonEmptyString(view.RecommendationTitle, view.Recommendation, view.NextRecommendedAction), "not available"),
		"Run ID: " + emptyAs(view.RunID, "not available"),
		"Report: " + emptyAs(view.ReportDir, "not available"),
	}
	if strings.EqualFold(view.Status, "not ready") || len(view.BlockingReasons) > 0 {
		lines = append(lines, "Repair readiness: Not ready")
		if len(view.BlockingReasons) > 0 {
			lines = append(lines, "Blocking reasons:")
			for _, reason := range view.BlockingReasons {
				lines = append(lines, "- "+reason)
			}
		}
	}
	if len(view.Errors) > 0 && !strings.EqualFold(view.Status, "not ready") {
		lines = append(lines, "Errors: "+strings.Join(view.Errors, "; "))
	}
	return safety.RedactString(strings.Join(lines, "\r\n"))
}

func extractStateSummary(state *GUIState, value any) {
	if value == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return
	}
	state.Health = firstString(decoded, "Health", "DetectionHealth", "InitialHealth", "FinalHealth", "Status", "Classification")
	state.InstallMode = firstString(decoded, "InstallMode", "InitialInstallMode", "FinalInstallMode")
	state.RecommendationTitle = firstString(decoded, "Recommendation", "Summary", "NextRecommendedAction", "NextAction")
	if status := firstString(decoded, "Status"); status == "not_ready" {
		state.CurrentStatus = "Not ready"
	}
	if ready, ok := boolField(decoded, "ready_for_repair", "ReadyForRepair"); ok && !ready {
		state.CurrentStatus = "Not ready"
	}
	if reportDir := firstString(decoded, "ReportDir"); reportDir != "" {
		state.LatestReportDir = reportDir
		state.Files.ReportDir = reportDir
	}
	state.Files.SupportBundlePath = firstString(decoded, "BundlePath", "SupportBundlePath")
	state.Files.CleanupPlanFile = nestedString(decoded, []string{"cleanup", "plan_path"}, []string{"repair_plan", "cleanup", "plan_path"})
	state.BlockingReasons = append(state.BlockingReasons, blockingReasonsFromDecoded(decoded)...)
	extractNestedStateSummary(state, decoded)
	if warnings := stringSlice(decoded, "Warnings"); len(warnings) > 0 {
		state.Warnings = append(state.Warnings, warnings...)
	}
	if errors := stringSlice(decoded, "Errors"); len(errors) > 0 {
		state.BlockingReasons = append(state.BlockingReasons, blockingReasonsFromErrors(errors)...)
	}
}

func extractNestedResultSummary(view *ResultView, decoded map[string]any) {
	if view == nil {
		return
	}
	if classification, ok := decoded["classification"].(map[string]any); ok {
		if view.Classification == "" {
			view.Classification = firstString(classification, "Health", "Status")
		}
		if issue, ok := classification["primary_issue"].(map[string]any); ok && view.PrimaryIssueCode == "" {
			view.PrimaryIssueCode = firstString(issue, "Code")
		}
	}
	if recommendation, ok := decoded["recommendation"].(map[string]any); ok {
		if action, ok := recommendation["primary_action"].(map[string]any); ok {
			if view.NextRecommendedAction == "" {
				view.NextRecommendedAction = firstString(action, "Code")
			}
			if view.Recommendation == "" {
				view.Recommendation = firstString(action, "Message", "Code")
			}
		}
		if view.Recommendation == "" {
			view.Recommendation = firstString(recommendation, "Status")
		}
	}
}

func extractNestedStateSummary(state *GUIState, decoded map[string]any) {
	for _, source := range []map[string]any{decoded, mapField(decoded, "repair_plan")} {
		if source == nil {
			continue
		}
		if state.Health == "" {
			state.Health = firstString(source, "DetectionHealth", "InitialHealth", "FinalHealth", "Health", "Status")
		}
		if state.InstallMode == "" {
			state.InstallMode = firstString(source, "InstallMode", "InitialInstallMode", "FinalInstallMode")
		}
		if classification := mapField(source, "classification"); classification != nil {
			if state.Health == "" {
				state.Health = firstString(classification, "Health", "Status")
			}
			if issue := mapField(classification, "primary_issue"); issue != nil {
				state.PrimaryIssueCode = firstString(issue, "Code")
				state.PrimaryIssueTitle = firstString(issue, "Title", "Message", "Description")
			}
		}
		if recommendation := mapField(source, "recommendation"); recommendation != nil {
			if action := mapField(recommendation, "primary_action"); action != nil {
				state.RecommendationCode = firstString(action, "Code")
				state.RecommendationTitle = firstString(action, "Title", "Message", "Description", "Code")
			} else if state.RecommendationTitle == "" {
				state.RecommendationTitle = firstString(recommendation, "Status")
			}
		}
		state.BlockingReasons = append(state.BlockingReasons, blockingReasonsFromDecoded(source)...)
	}
}

func resultViewFromGUIState(state GUIState) ResultView {
	return ResultView{
		RunID:                 state.LatestRunID,
		Status:                state.CurrentStatus,
		Workflow:              state.CurrentWorkflow,
		ReportDir:             state.Files.ReportDir,
		SummaryFile:           state.Files.SummaryFile,
		OperationsFile:        state.Files.OperationsFile,
		PrimaryResultFile:     state.Files.PrimaryResultFile,
		SupportBundlePath:     state.Files.SupportBundlePath,
		Classification:        state.Health,
		InstallMode:           state.InstallMode,
		PrimaryIssueCode:      state.PrimaryIssueCode,
		PrimaryIssueTitle:     state.PrimaryIssueTitle,
		Recommendation:        state.RecommendationTitle,
		RecommendationCode:    state.RecommendationCode,
		RecommendationTitle:   state.RecommendationTitle,
		NextRecommendedAction: state.RecommendationCode,
		BlockingReasons:       state.BlockingReasons,
		Warnings:              state.Warnings,
	}
}

func firstString(decoded map[string]any, keys ...string) string {
	for _, key := range keys {
		for _, candidate := range []string{key, strings.ToLower(key[:1]) + key[1:], camelToSnake(key)} {
			if value, ok := decoded[candidate].(string); ok {
				return safety.RedactString(value)
			}
		}
	}
	return ""
}

func stringSlice(decoded map[string]any, keys ...string) []string {
	for _, key := range keys {
		for _, candidate := range []string{key, strings.ToLower(key[:1]) + key[1:], camelToSnake(key)} {
			raw, ok := decoded[candidate].([]any)
			if !ok {
				continue
			}
			values := make([]string, 0, len(raw))
			for _, item := range raw {
				if value, ok := item.(string); ok {
					values = append(values, safety.RedactString(value))
				}
			}
			return values
		}
	}
	return nil
}

func AggregateTimelineWarnings(items []TimelineItem, warnings []string, detailsFile string) []TimelineItem {
	counts := map[string]int{}
	for _, item := range items {
		key := warningAggregationKey(item.Message)
		if key != "" {
			counts[key]++
		}
	}
	for _, warning := range warnings {
		key := warningAggregationKey(warning)
		if key != "" {
			counts[key]++
		}
	}
	filtered := make([]TimelineItem, 0, len(items)+len(counts))
	for _, item := range items {
		if warningAggregationKey(item.Message) == "" {
			filtered = append(filtered, item)
		}
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		message := fmt.Sprintf("%s: %d.", key, counts[key])
		filtered = append(filtered, TimelineItem{
			Step:        "warning-summary",
			Status:      string(app.OperationStatusWarning),
			Message:     message,
			DetailsFile: detailsFile,
		})
	}
	return filtered
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

func supportStatus(status string, result any, errors []string) string {
	if isResultNotReady(result) {
		return "Not ready"
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return "Success"
	case "warning":
		return "Warning"
	case "failed":
		if len(errors) > 0 && hasNotReadyError(errors) {
			return "Not ready"
		}
		return "Failed"
	case "cancelled", "canceled":
		return "Cancelled"
	case "running":
		return "Running"
	case "", "idle":
		return "Idle"
	default:
		return status
	}
}

func isResultNotReady(value any) bool {
	data, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return false
	}
	if strings.EqualFold(firstString(decoded, "Status"), "not_ready") {
		return true
	}
	if ready, ok := boolField(decoded, "ready_for_repair", "ReadyForRepair"); ok && !ready {
		return true
	}
	if plan := mapField(decoded, "repair_plan"); plan != nil {
		if strings.EqualFold(firstString(plan, "Status"), "not_ready") {
			return true
		}
		if ready, ok := boolField(plan, "ready_for_repair", "ReadyForRepair"); ok && !ready {
			return true
		}
	}
	return false
}

func hasNotReadyError(errors []string) bool {
	for _, err := range errors {
		normalized := strings.ToLower(err)
		if strings.Contains(normalized, "not_ready") || strings.Contains(normalized, "not ready") || strings.Contains(normalized, "admin_rights") || strings.Contains(normalized, "installer") {
			return true
		}
	}
	return false
}

func blockingReasonsFromDecoded(decoded map[string]any) []string {
	var reasons []string
	for _, source := range []map[string]any{decoded, mapField(decoded, "preflight")} {
		if source == nil {
			continue
		}
		if checks, ok := source["checks"].([]any); ok {
			for _, raw := range checks {
				check, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				required, _ := boolField(check, "required", "Required")
				status := strings.ToLower(firstString(check, "Status", "status"))
				if !required || (status != "fail" && status != "failed") {
					continue
				}
				reasons = append(reasons, readableBlocker(firstString(check, "Code", "Name", "code", "name"), firstString(check, "Title", "Message", "Error", "Action")))
			}
		}
		if failed := stringSlice(source, "FailedRequiredChecks"); len(failed) > 0 {
			for _, code := range failed {
				reasons = append(reasons, readableBlocker(code, ""))
			}
		}
		if blocked := stringSlice(source, "BlockedByPolicy", "BlockingPolicyChecks"); len(blocked) > 0 {
			for _, code := range blocked {
				reasons = append(reasons, readableBlocker(code, ""))
			}
		}
	}
	if installer := mapField(decoded, "installer"); installer != nil {
		if found, ok := boolField(installer, "installer_found", "exists", "Exists"); ok && !found {
			reasons = append(reasons, "Installer file was not found.")
		}
	}
	if repairPlan := mapField(decoded, "repair_plan"); repairPlan != nil {
		reasons = append(reasons, blockingReasonsFromDecoded(repairPlan)...)
	}
	return reasons
}

func blockingReasonsFromErrors(errors []string) []string {
	reasons := make([]string, 0, len(errors))
	for _, errText := range errors {
		reasons = append(reasons, readableBlocker(errText, errText))
	}
	return reasons
}

func readableBlocker(code string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(code + " " + fallback))
	switch {
	case strings.Contains(normalized, "admin_rights"), strings.Contains(normalized, "administrator"):
		return "Administrator rights required."
	case strings.Contains(normalized, "installer_available"), strings.Contains(normalized, "installer path is required"), strings.Contains(normalized, "installer file"), strings.Contains(normalized, "installer not found"), strings.Contains(normalized, "file missing"):
		return "Installer file was not found."
	case strings.Contains(normalized, "invite_present"), strings.Contains(normalized, "invite is required"):
		return "Invite is required for this workflow."
	case strings.Contains(normalized, "preflight_not_ready"), strings.Contains(normalized, "not_ready"), strings.Contains(normalized, "not ready"):
		return "Repair is not ready. Resolve blocking checks and run Preflight again."
	case strings.Contains(normalized, "defender_status_unavailable"):
		return "Defender status could not be verified. Run as Administrator or check Defender manually."
	case strings.TrimSpace(fallback) != "":
		return safety.RedactString(fallback)
	default:
		return safety.RedactString(code)
	}
}

func mapField(decoded map[string]any, key string) map[string]any {
	if decoded == nil {
		return nil
	}
	for _, candidate := range []string{key, strings.ToLower(key[:1]) + key[1:], camelToSnake(key)} {
		if value, ok := decoded[candidate].(map[string]any); ok {
			return value
		}
	}
	return nil
}

func boolField(decoded map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		for _, candidate := range []string{key, strings.ToLower(key[:1]) + key[1:], camelToSnake(key)} {
			if value, ok := decoded[candidate].(bool); ok {
				return value, true
			}
		}
	}
	return false, false
}

func nestedString(decoded map[string]any, paths ...[]string) string {
	for _, path := range paths {
		current := decoded
		for i, part := range path {
			if i == len(path)-1 {
				if value := firstString(current, part); value != "" {
					return value
				}
				continue
			}
			current = mapField(current, part)
			if current == nil {
				break
			}
		}
	}
	return ""
}

func dedupeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(safety.RedactString(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func supportIcon(status string) string {
	switch strings.ToLower(status) {
	case "success", "pass":
		return "OK"
	case "warning":
		return "WARN"
	case "failed", "fail":
		return "FAIL"
	case "skipped":
		return "SKIP"
	default:
		return "INFO"
	}
}

func supportMessage(message string) string {
	message = strings.TrimSpace(safety.RedactString(message))
	if message == "" {
		return "Operation completed"
	}
	return message
}

func detailsFileForAggregation(view ResultView) string {
	if view.ReportDir == "" {
		return ""
	}
	candidate := filepath.Join(view.ReportDir, "cleanup-plan.json")
	if view.PrimaryResultFile != "" && strings.Contains(strings.ToLower(view.PrimaryResultFile), "cleanup") {
		return view.PrimaryResultFile
	}
	return candidate
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

func redactSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, safety.RedactString(value))
	}
	return out
}

func emptyAs(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return safety.RedactString(value)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
