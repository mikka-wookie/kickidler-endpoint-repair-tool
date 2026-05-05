package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var requiredDocs = []string{
	"SUPPORT-KB.md",
	"COMMAND-REFERENCE.md",
	"REPORT-FILES.md",
	"CLASSIFICATIONS.md",
	"EXIT-CODES.md",
	"TROUBLESHOOTING.md",
	"ESCALATION-CHECKLIST.md",
	"SAFETY-MODEL.md",
	"RELEASE-CHECKLIST.md",
}

func readDoc(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

func TestRequiredSupportDocsExistAndUseInvitePlaceholder(t *testing.T) {
	for _, name := range requiredDocs {
		t.Run(name, func(t *testing.T) {
			content := readDoc(t, name)
			if strings.Contains(content, "VERY_SECRET_INVITE_SHOULD_NOT_APPEAR") {
				t.Fatalf("%s contains fake secret marker", name)
			}
			if !strings.Contains(content, "<INVITE>") {
				t.Fatalf("%s should use <INVITE> placeholder", name)
			}
		})
	}
}

func TestCommandDocsMentionSafetyDistinction(t *testing.T) {
	content := readDoc(t, "COMMAND-REFERENCE.md")
	for _, want := range []string{
		"Read-only / non-destructive",
		"System-modifying / destructive",
		"repair --dry-run",
		"repair --yes",
		"reports cleanup --dry-run",
		"reports cleanup --yes",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("COMMAND-REFERENCE.md missing %q", want)
		}
	}
}

func TestExitCodeDocsMentionStandardCodes(t *testing.T) {
	content := readDoc(t, "EXIT-CODES.md")
	for _, want := range []string{"`0`", "`1`", "`7`", "`10`"} {
		if !strings.Contains(content, want) {
			t.Fatalf("EXIT-CODES.md missing %q", want)
		}
	}
}

func TestSafetyModelMentionsValidatedCleanupPlan(t *testing.T) {
	content := strings.ToLower(readDoc(t, "SAFETY-MODEL.md"))
	for _, want := range []string{"validated cleanup plan", "must never delete"} {
		if !strings.Contains(content, want) {
			t.Fatalf("SAFETY-MODEL.md missing %q", want)
		}
	}
}

func TestSupportKBMentionsSupportBundleAndReportPath(t *testing.T) {
	content := readDoc(t, "SUPPORT-KB.md")
	for _, want := range []string{
		`C:\ProgramData\kigrepair\Reports\<timestamp>\`,
		"kigrepair-support-bundle.zip",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("SUPPORT-KB.md missing %q", want)
		}
	}
}

func TestReleaseChecklistMentionsGoTest(t *testing.T) {
	content := readDoc(t, "RELEASE-CHECKLIST.md")
	if !strings.Contains(content, "go test ./...") {
		t.Fatal("RELEASE-CHECKLIST.md should mention go test ./...")
	}
}
