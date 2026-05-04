package verifier

import (
	"testing"

	"kigrepair/internal/detector"
)

func TestVerifyHealthyReportSucceeds(t *testing.T) {
	got := Verify(healthyReport(), Options{})
	if got.Status != VerificationSuccess {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationSuccess, got)
	}
}

func TestVerifyMissingDefenderWarns(t *testing.T) {
	report := healthyReport()
	report.MissingDefenderPaths = []string{`C:\Program Files\TeleLinkSoft\bin`}
	got := Verify(report, Options{})
	if got.Status != VerificationWarning {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationWarning, got)
	}
}

func TestVerifyBrokenHealthFails(t *testing.T) {
	report := healthyReport()
	report.Health = detector.GrabberHealthBroken
	report.ServiceExecutableExists = false
	got := Verify(report, Options{})
	if got.Status != VerificationFailed {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationFailed, got)
	}
}

func TestVerifyInstallLogRequiredOnlyWhenInstallExecuted(t *testing.T) {
	got := Verify(healthyReport(), Options{InstallExecuted: true, MSIInstallLog: `C:\missing\msi-install.log`})
	if got.Status != VerificationFailed {
		t.Fatalf("status = %s, want %s: %#v", got.Status, VerificationFailed, got)
	}
}

func healthyReport() detector.DetectionReport {
	return detector.DetectionReport{
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft\bin`,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutableExists: true,
		RequiredDefenderPaths:   []string{`C:\Program Files\TeleLinkSoft\bin`},
		Services: []detector.ServiceState{
			{Name: "ngs", Exists: true, Status: "running"},
		},
		Processes: []detector.ProcessState{
			{Name: "grabber2.exe", PID: 100, ExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, MatchedByName: true},
		},
		Defender: detector.DefenderState{Available: true},
	}
}
