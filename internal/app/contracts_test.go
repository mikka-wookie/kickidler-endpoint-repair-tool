package app

import (
	"encoding/json"
	"strings"
	"testing"
)

const secretInvite = "REAL-SECRET-INVITE"

func TestInstallerRequestInviteIsNotSerialized(t *testing.T) {
	req := RepairRequest{
		CommonRequest: CommonRequest{Profile: "standard"},
		InstallerRequestFields: InstallerRequestFields{
			InstallerPath: `C:\Temp\grabber.msi`,
			InviteValue:   secretInvite,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secretInvite) {
		t.Fatalf("serialized request leaked invite: %s", data)
	}
}

func TestSafeRequestSummaryRedactsInvite(t *testing.T) {
	req := PreflightRequest{InstallerRequestFields: InstallerRequestFields{InviteValue: secretInvite}}
	summary := SafeRequestSummary(req)
	if got := summary["invite"]; got != "<REDACTED>" {
		t.Fatalf("invite summary = %#v, want redacted", got)
	}
	data, _ := json.Marshal(summary)
	if strings.Contains(string(data), secretInvite) {
		t.Fatalf("sanitized summary leaked invite: %s", data)
	}
}

func TestProgressAndOperationRedaction(t *testing.T) {
	event := RedactProgressEvent(ProgressEvent{Message: "invite=" + secretInvite})
	if strings.Contains(event.Message, secretInvite) {
		t.Fatalf("progress event leaked invite: %#v", event)
	}
	result := RedactOperationResult(OperationResult{Message: "invite=" + secretInvite, Error: "secret: " + secretInvite})
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), secretInvite) {
		t.Fatalf("operation result leaked invite: %s", data)
	}
}

func TestResponseJSONDoesNotContainInvite(t *testing.T) {
	response := WorkflowResponse{
		Meta: WorkflowResponseMeta{RunID: "run-1", Workflow: "repair", Command: "repair", Status: string(WorkflowStatusFailed), ExitCode: ExitInvalidInput},
		Result: map[string]any{
			"invite_present": true,
			"command":        `msiexec /i "grabber.msi" invite=<REDACTED>`,
		},
	}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secretInvite) {
		t.Fatalf("response leaked invite: %s", data)
	}
}
