package installer

import (
	"testing"

	"kigrepair/internal/detector"
)

func TestVerifyInstall(t *testing.T) {
	msiSuccess := ClassifyMSIInstallExitCode(0)
	tests := []struct {
		name string
		msi  MSIResult
		rep  detector.DetectionReport
		want VerificationStatus
	}{
		{name: "msi success final healthy", msi: msiSuccess, rep: healthyReport(false), want: VerificationSuccess},
		{name: "msi success defender missing", msi: msiSuccess, rep: healthyReport(true), want: VerificationWarning},
		{name: "msi success final broken", msi: msiSuccess, rep: brokenReport(), want: VerificationFailed},
		{name: "msi failed", msi: ClassifyMSIInstallExitCode(1603), rep: healthyReport(false), want: VerificationFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyInstall(tt.msi, tt.rep)
			if got.Status != tt.want {
				t.Fatalf("status = %s, want %s: %#v", got.Status, tt.want, got)
			}
		})
	}
}

func healthyReport(defenderMissing bool) detector.DetectionReport {
	report := detector.DetectionReport{
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft\bin`,
		PrimaryService:          "ngs",
		PrimaryServiceImagePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutableExists: true,
		RequiredDefenderPaths:   []string{`C:\Program Files\TeleLinkSoft\bin`},
		Services: []detector.ServiceState{
			{Name: "ngs", Exists: true, Status: "running"},
		},
		Defender: detector.DefenderState{Available: true, ExclusionPaths: []string{`C:\Program Files\TeleLinkSoft\bin`}},
	}
	if defenderMissing {
		report.MissingDefenderPaths = []string{`C:\Program Files\TeleLinkSoft\bin`}
	}
	return report
}

func brokenReport() detector.DetectionReport {
	report := healthyReport(false)
	report.Health = detector.GrabberHealthBroken
	report.ServiceExecutableExists = false
	return report
}
