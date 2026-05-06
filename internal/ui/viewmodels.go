package ui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"kigrepair/internal/app"
	"kigrepair/internal/safety"
)

func BuildResultView(resp *app.WorkflowResponse) ResultView {
	if resp == nil {
		return ResultView{Status: string(app.WorkflowStatusFailed), Errors: []string{"workflow returned no response"}}
	}
	view := ResultView{
		RunID:             resp.Meta.RunID,
		Status:            resp.Meta.Status,
		Workflow:          resp.Meta.Workflow,
		ReportDir:         resp.Meta.ReportDir,
		SummaryFile:       resp.Meta.SummaryFile,
		OperationsFile:    resp.Meta.OperationsFile,
		PrimaryResultFile: resp.Meta.PrimaryResultFile,
		Warnings:          redactSlice(resp.Warnings),
		Errors:            redactSlice(resp.Errors),
	}
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
		})
	}
	extractResultSummary(&view, resp.Result)
	if view.SupportBundlePath == "" && view.ReportDir != "" && resp.Meta.Workflow == "collect-report" {
		view.SupportBundlePath = filepath.Join(view.ReportDir, "kigrepair-support-bundle.zip")
	}
	return view
}

func FormatTimeline(items []TimelineItem) string {
	if len(items) == 0 {
		return "No timeline events yet."
	}
	var b strings.Builder
	for _, item := range items {
		status := item.Status
		if status == "" {
			status = "info"
		}
		label := item.Step
		if label == "" {
			label = item.Stage
		}
		if label == "" {
			label = "workflow"
		}
		fmt.Fprintf(&b, "[%s] %s: %s", status, label, item.Message)
		if item.Target != "" {
			fmt.Fprintf(&b, " (%s)", item.Target)
		}
		if item.DurationMS > 0 {
			fmt.Fprintf(&b, " in %d ms", item.DurationMS)
		}
		if item.FailureCategory != "" && strings.EqualFold(item.Status, string(app.OperationStatusFailed)) {
			fmt.Fprintf(&b, " - %s", item.FailureCategory)
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
		lines = append(lines, "Primary issue: "+view.PrimaryIssueCode)
	}
	if view.Recommendation != "" {
		lines = append(lines, "Recommendation: "+view.Recommendation)
	} else if view.NextRecommendedAction != "" {
		lines = append(lines, "Recommendation: "+view.NextRecommendedAction)
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
