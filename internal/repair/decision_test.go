package repair

import (
	"testing"

	"kigrepair/internal/detector"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name           string
		report         detector.DetectionReport
		force          bool
		cleanupActions int
		wantCleanup    bool
		wantInstall    bool
		wantDefender   bool
		wantStop       bool
	}{
		{
			name:         "healthy no force",
			report:       reportWithHealth(detector.GrabberHealthHealthy, nil),
			wantCleanup:  false,
			wantInstall:  false,
			wantDefender: false,
		},
		{
			name:         "healthy missing defender",
			report:       reportWithHealth(detector.GrabberHealthHealthy, []string{`C:\Program Files\TeleLinkSoft\bin`}),
			wantCleanup:  false,
			wantInstall:  false,
			wantDefender: true,
		},
		{
			name:         "healthy force",
			report:       reportWithHealth(detector.GrabberHealthHealthy, nil),
			force:        true,
			wantCleanup:  true,
			wantInstall:  true,
			wantDefender: true,
		},
		{
			name:         "broken",
			report:       reportWithHealth(detector.GrabberHealthBroken, nil),
			wantCleanup:  true,
			wantInstall:  true,
			wantDefender: true,
		},
		{
			name:         "partially removed",
			report:       reportWithHealth(detector.GrabberHealthPartiallyRemoved, nil),
			wantCleanup:  true,
			wantInstall:  true,
			wantDefender: true,
		},
		{
			name:           "not installed with no cleanup actions",
			report:         reportWithHealth(detector.GrabberHealthNotInstalled, nil),
			cleanupActions: 0,
			wantCleanup:    false,
			wantInstall:    true,
			wantDefender:   true,
		},
		{
			name:           "not installed with cleanup actions",
			report:         reportWithHealth(detector.GrabberHealthNotInstalled, nil),
			cleanupActions: 1,
			wantCleanup:    true,
			wantInstall:    true,
			wantDefender:   true,
		},
		{
			name:     "unknown no force",
			report:   reportWithHealth(detector.GrabberHealthUnknown, nil),
			wantStop: true,
		},
		{
			name:         "unknown force",
			report:       reportWithHealth(detector.GrabberHealthUnknown, nil),
			force:        true,
			wantCleanup:  true,
			wantInstall:  true,
			wantDefender: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.report, tt.force, tt.cleanupActions)
			if got.CleanupNeeded != tt.wantCleanup || got.InstallNeeded != tt.wantInstall || got.DefenderNeeded != tt.wantDefender || got.Stop != tt.wantStop {
				t.Fatalf("Decide() = cleanup=%t install=%t defender=%t stop=%t, want cleanup=%t install=%t defender=%t stop=%t",
					got.CleanupNeeded, got.InstallNeeded, got.DefenderNeeded, got.Stop,
					tt.wantCleanup, tt.wantInstall, tt.wantDefender, tt.wantStop)
			}
		})
	}
}

func reportWithHealth(health detector.GrabberHealthStatus, missingDefender []string) detector.DetectionReport {
	return detector.DetectionReport{
		Health:               health,
		InstallMode:          detector.InstallModeStandard,
		InstallRoot:          `C:\Program Files\TeleLinkSoft\bin`,
		MissingDefenderPaths: missingDefender,
		Defender: detector.DefenderState{
			Available:    true,
			MissingPaths: missingDefender,
		},
	}
}
