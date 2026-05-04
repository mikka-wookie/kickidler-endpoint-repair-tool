package recommendations

import "kigrepair/internal/classifier"

func FromClassifier(result classifier.ClassificationResult) ClassificationResult {
	converted := ClassificationResult{
		Status: result.Status,
		Issues: make([]ClassifiedIssue, 0, len(result.Issues)),
	}
	for _, issue := range result.Issues {
		converted.Issues = append(converted.Issues, ClassifiedIssue{
			Code:        issue.Code,
			Title:       issue.Title,
			Description: issue.Description,
			Severity:    issue.Severity,
		})
	}
	if result.PrimaryIssue != nil {
		converted.PrimaryIssue = &ClassifiedIssue{
			Code:        result.PrimaryIssue.Code,
			Title:       result.PrimaryIssue.Title,
			Description: result.PrimaryIssue.Description,
			Severity:    result.PrimaryIssue.Severity,
		}
	}
	return converted
}
