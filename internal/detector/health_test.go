package detector

import "testing"

func TestCalculateHealthHealthyWithUnavailableDefenderWarning(t *testing.T) {
	report := baseReport()
	report.Services = []ServiceState{{
		Name:                   wmiProviderService,
		Exists:                 true,
		Status:                 "running",
		ImagePath:              `C:\Windows\System32\wmi\bin\svchost.exe`,
		ExpectedImagePathMatch: true,
	}}
	report.Files = requiredExistingFiles(report.System)
	report.Defender = DefenderState{Available: false}

	health, issues, recommendations := CalculateHealth(report)

	if health != GrabberHealthHealthy {
		t.Fatalf("health = %s, want %s", health, GrabberHealthHealthy)
	}
	if !containsString(issues, "Defender exclusion could not be verified") {
		t.Fatalf("expected Defender verification issue, got %#v", issues)
	}
	if !containsString(recommendations, "Verify Defender exclusions manually") {
		t.Fatalf("expected Defender recommendation, got %#v", recommendations)
	}
}

func TestCalculateHealthBrokenWhenServiceExistsAndRequiredFileMissing(t *testing.T) {
	report := baseReport()
	report.Services = []ServiceState{{
		Name:                   wmiProviderService,
		Exists:                 true,
		Status:                 "stopped",
		ImagePath:              `C:\Windows\System32\wmi\bin\svchost.exe`,
		ExpectedImagePathMatch: true,
	}}
	report.Files = []FileState{
		{Path: `C:\Windows\System32\wmi\bin\svchost.exe`, Exists: false, Type: "file"},
		{Path: `C:\Windows\System32\wmi\bin\WmiPrvSE.exe`, Exists: true, Type: "file"},
		{Path: `C:\Windows\System32\wmi\bin\RuntimeBroker.exe`, Exists: true, Type: "file"},
	}
	report.Defender = DefenderState{Available: true, MissingPaths: []string{`C:\Windows\System32\wmi`}}

	health, issues, _ := CalculateHealth(report)

	if health != GrabberHealthBroken {
		t.Fatalf("health = %s, want %s", health, GrabberHealthBroken)
	}
	if !containsString(issues, `Missing required file: C:\Windows\System32\wmi\bin\svchost.exe`) {
		t.Fatalf("expected missing file issue, got %#v", issues)
	}
}

func TestCalculateHealthPartiallyRemovedForRegistryLeftovers(t *testing.T) {
	report := baseReport()
	report.Registry = []RegistryState{{Root: "HKLM", Path: `SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`, Exists: true}}

	health, issues, _ := CalculateHealth(report)

	if health != GrabberHealthPartiallyRemoved {
		t.Fatalf("health = %s, want %s", health, GrabberHealthPartiallyRemoved)
	}
	if !containsString(issues, "Registry leftovers detected without service") {
		t.Fatalf("expected registry leftovers issue, got %#v", issues)
	}
}

func TestCalculateHealthNotInstalled(t *testing.T) {
	report := baseReport()

	health, issues, recommendations := CalculateHealth(report)

	if health != GrabberHealthNotInstalled {
		t.Fatalf("health = %s, want %s", health, GrabberHealthNotInstalled)
	}
	if len(issues) != 0 || len(recommendations) != 0 {
		t.Fatalf("expected no issues/recommendations, got %#v %#v", issues, recommendations)
	}
}

func TestMissingPathsNormalizesCaseAndSlashes(t *testing.T) {
	missing := MissingPaths(
		[]string{`C:\Program Files\TeleLinkSoft`, `C:\Windows\System32\wmi`},
		[]string{`c:/program files/telelinksoft`},
	)

	if len(missing) != 1 || missing[0] != `C:\Windows\System32\wmi` {
		t.Fatalf("missing = %#v", missing)
	}
}

func TestPathsEqualNormalizesQuotesAndCase(t *testing.T) {
	if !pathsEqual(`"C:\Windows\System32\wmi\bin\svchost.exe"`, `c:/windows/system32/wmi/bin/SVCHOST.EXE`) {
		t.Fatal("expected paths to match")
	}
}

func TestPathsEqualNormalizesWindowsEnvironmentVariables(t *testing.T) {
	t.Setenv("SystemRoot", `C:\Windows`)
	if !pathsEqual(`%SystemRoot%\System32\wmi\bin\svchost.exe`, `C:\Windows\System32\wmi\bin\svchost.exe`) {
		t.Fatal("expected environment-expanded paths to match")
	}
}

func baseReport() DetectionReport {
	return DetectionReport{
		System: SystemState{
			SystemRoot:   `C:\Windows`,
			ProgramData:  `C:\ProgramData`,
			ProgramFiles: `C:\Program Files`,
		},
		Services: []ServiceState{
			{Name: "ngs", Exists: false},
			{Name: "tls", Exists: false},
			{Name: wmiProviderService, Exists: false},
		},
		Defender: DefenderState{Available: true},
	}
}

func requiredExistingFiles(system SystemState) []FileState {
	files := make([]FileState, 0, 3)
	for _, path := range requiredWMIFilePaths(system) {
		files = append(files, FileState{Path: path, Exists: true, Type: "file"})
	}
	return files
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
