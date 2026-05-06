package cleaner

import (
	"errors"
	"testing"
	"time"

	"kigrepair/internal/config"
	"kigrepair/internal/detector"
)

func TestBuildPlanPlansServiceActions(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{
		Services: []detector.ServiceState{{Name: "ngs", Exists: true, Status: "running", TrustLevel: "trusted", NormalizedExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`}},
	}, testPlanOptions(nil, nil))

	assertAction(t, plan, CleanupActionStopService, "ngs")
	assertAction(t, plan, CleanupActionDeleteService, "ngs")
}

func TestBuildPlanSkipsPathMismatchServiceActions(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{
		Services: []detector.ServiceState{{Name: "ngs", Exists: true, TrustLevel: "path_mismatch", NormalizedExecutablePath: `C:\Unexpected\service.exe`}},
	}, testPlanOptions(nil, nil))

	assertNoAction(t, plan, CleanupActionStopService, "ngs")
	assertNoAction(t, plan, CleanupActionDeleteService, "ngs")
	if len(plan.Warnings) != 1 {
		t.Fatalf("expected warning for skipped service, got %#v", plan.Warnings)
	}
}

func TestBuildPlanPlansProcessAction(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{
		Processes: []detector.ProcessState{{
			Name:           "grabber2.exe",
			PID:            1234,
			ExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
			GrabberRelated: true,
			TrustLevel:     detector.ProcessTrustNameAndPathMatch,
			CanTerminate:   true,
		}},
	}, testPlanOptions(nil, nil))

	assertAction(t, plan, CleanupActionKillProcess, `grabber2.exe PID 1234 C:\Program Files\TeleLinkSoft\bin\grabber2.exe`)
}

func TestBuildPlanSkipsUnsafeProcessAction(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{
		Processes: []detector.ProcessState{{
			Name:           "svchost.exe",
			PID:            888,
			ExecutablePath: `C:\Windows\System32\svchost.exe`,
			TrustLevel:     detector.ProcessTrustPathMismatch,
			MatchReason:    "normal Windows process, not Grabber hidden WMI path",
		}},
	}, testPlanOptions(nil, nil))

	assertNoAction(t, plan, CleanupActionKillProcess, `svchost.exe PID 888 C:\Windows\System32\svchost.exe`)
	if len(plan.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want one skipped process warning", plan.Warnings)
	}
}

func TestBuildPlanPlansExistingFolderAction(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{}, testPlanOptions(map[string]bool{
		`C:\Program Files\TeleLinkSoft`: true,
	}, nil))

	assertAction(t, plan, CleanupActionDeletePath, `C:\Program Files\TeleLinkSoft`)
}

func TestBuildPlanSkipsMissingFolderAction(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{}, testPlanOptions(map[string]bool{
		`C:\Program Files\TeleLinkSoft`: false,
	}, nil))

	assertNoAction(t, plan, CleanupActionDeletePath, `C:\Program Files\TeleLinkSoft`)
}

func TestBuildPlanAddsBlockerForUnsafeExistingPath(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{}, testPlanOptions(map[string]bool{
		`C:\Unsafe`: true,
	}, errors.New("unsafe path")))

	if len(plan.Blockers) != 1 {
		t.Fatalf("expected one blocker, got %d: %#v", len(plan.Blockers), plan.Blockers)
	}
	assertNoAction(t, plan, CleanupActionDeletePath, `C:\Unsafe`)
}

func TestBuildPlanPlansRegistryActions(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{
		Registry: []detector.RegistryState{{
			Root:   "HKLM",
			Path:   `SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`,
			Exists: true,
		}},
	}, testPlanOptions(nil, nil))

	assertAction(t, plan, CleanupActionDeleteRegistryKey, `HKLM\SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`)
}

func TestBuildPlanPlansMSIUninstallWhenMSIRegistryExists(t *testing.T) {
	plan := BuildPlan(detector.DetectionReport{
		Registry: []detector.RegistryState{{
			Root:   "HKCR",
			Path:   `Installer\Products\73CBF1BE79B05FC43A92FC82AB567384`,
			Exists: true,
		}},
	}, testPlanOptions(nil, nil))

	assertAction(t, plan, CleanupActionMSIUninstall, `{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`)
}

func TestConservativePolicySkipsHiddenWMIPathCleanup(t *testing.T) {
	cfg, _ := config.ConfigForProfile("conservative")
	policy := config.EffectiveConfig{Config: cfg}.PolicySummary()
	opts := testPlanOptions(map[string]bool{
		`C:\Windows\System32\wmi`: true,
	}, nil)
	opts.Policy = &policy
	plan := BuildPlan(detector.DetectionReport{}, opts)

	assertNoAction(t, plan, CleanupActionDeletePath, `C:\Windows\System32\wmi`)
	if len(plan.Skipped) != 1 || plan.Skipped[0].PolicyStatus != "skipped_by_policy" {
		t.Fatalf("expected hidden WMI path skipped by policy, got %#v", plan.Skipped)
	}
}

func testPlanOptions(existing map[string]bool, validateErr error) PlanOptions {
	if existing == nil {
		existing = map[string]bool{}
	}
	cleanupPaths := make([]string, 0, len(existing))
	for path := range existing {
		cleanupPaths = append(cleanupPaths, path)
	}
	return PlanOptions{
		DryRun:       true,
		CleanupPaths: cleanupPaths,
		PathExists: func(path string) (bool, error) {
			return existing[path], nil
		},
		ValidatePath: func(string) error {
			return validateErr
		},
		Now: func() time.Time {
			return time.Date(2026, 5, 3, 14, 30, 22, 0, time.UTC)
		},
	}
}

func assertAction(t *testing.T, plan CleanupPlan, actionType CleanupActionType, target string) {
	t.Helper()
	for _, action := range plan.Actions {
		if action.Type == actionType && action.Target == target {
			return
		}
	}
	t.Fatalf("missing action type=%s target=%q in %#v", actionType, target, plan.Actions)
}

func assertNoAction(t *testing.T, plan CleanupPlan, actionType CleanupActionType, target string) {
	t.Helper()
	for _, action := range plan.Actions {
		if action.Type == actionType && action.Target == target {
			t.Fatalf("unexpected action type=%s target=%q in %#v", actionType, target, plan.Actions)
		}
	}
}
