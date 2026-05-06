package cliadapter

import (
	"bytes"
	"strings"
	"testing"

	"kigrepair/internal/app"
)

func TestCommonRequestMapsGlobalOptions(t *testing.T) {
	req := CommonRequest(GlobalOptions{
		Output:         `C:\Reports`,
		ConfigPath:     `C:\kigrepair.yaml`,
		Profile:        "conservative",
		Quiet:          true,
		NonInteractive: true,
		JSONOutput:     true,
		Yes:            true,
		DryRun:         true,
		LogLevel:       "debug",
	})
	if req.OutputDir != `C:\Reports` || req.Profile != "conservative" || !req.Quiet || !req.JSONOutput || !req.Yes || !req.DryRun {
		t.Fatalf("request mapping mismatch: %#v", req)
	}
}

func TestRepairRequestMapsInstallerInviteAndDoesNotSerializeInvite(t *testing.T) {
	req := RepairRequest(GlobalOptions{Profile: "standard"}, `C:\grabber.msi`, "REAL-SECRET-INVITE")
	if req.Profile != "standard" || req.InstallerPath != `C:\grabber.msi` || req.InviteValue == "" {
		t.Fatalf("repair request mapping mismatch: %#v", req)
	}
	var out bytes.Buffer
	if err := WriteJSON(&out, req); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "REAL-SECRET-INVITE") {
		t.Fatalf("JSON output leaked invite: %s", out.String())
	}
}

func TestExitErrorUsesResponseExitCode(t *testing.T) {
	err := ExitError(&app.WorkflowResponse{Meta: app.WorkflowResponseMeta{ExitCode: app.ExitInvalidInput}})
	exitErr, ok := err.(app.ExitError)
	if !ok || exitErr.Code != app.ExitInvalidInput {
		t.Fatalf("exit error = %#v", err)
	}
	if err := ExitError(&app.WorkflowResponse{Meta: app.WorkflowResponseMeta{ExitCode: app.ExitSuccess}}); err != nil {
		t.Fatalf("success response returned error: %v", err)
	}
}
