package detector

import (
	"strings"
	"testing"
)

func TestCalculateHealthHealthyStandardInstall(t *testing.T) {
	report := standardTLSReport(true, []string{`C:\Program Files\TeleLinkSoft`})

	health, _, _ := CalculateHealth(report)

	if health != GrabberHealthHealthy {
		t.Fatalf("health = %s, want %s", health, GrabberHealthHealthy)
	}
	enriched := EnrichDetectionReport(report)
	if enriched.InstallMode != InstallModeStandard {
		t.Fatalf("mode = %s, want %s", enriched.InstallMode, InstallModeStandard)
	}
}

func TestCalculateHealthStandardInstallMissingDefenderExclusionStillHealthy(t *testing.T) {
	report := standardTLSReport(true, nil)

	health, issues, _ := CalculateHealth(report)

	if health != GrabberHealthHealthy {
		t.Fatalf("health = %s, want %s", health, GrabberHealthHealthy)
	}
	if !containsString(issues, `Missing Defender exclusion: C:\Program Files\TeleLinkSoft`) {
		t.Fatalf("expected missing Defender issue, got %#v", issues)
	}
}

func TestCalculateHealthBrokenStandardInstallMissingServiceExecutable(t *testing.T) {
	report := standardTLSReport(false, []string{`C:\Program Files\TeleLinkSoft`})

	health, issues, _ := CalculateHealth(report)

	if health != GrabberHealthBroken {
		t.Fatalf("health = %s, want %s", health, GrabberHealthBroken)
	}
	if !containsString(issues, `Missing required file: C:\Program Files\TeleLinkSoft\tlsservice.exe`) {
		t.Fatalf("expected missing service executable issue, got %#v", issues)
	}
	if containsSubstring(issues, `C:\Windows\System32\wmi\bin`) {
		t.Fatalf("standard install should not mention WMI required files, got %#v", issues)
	}
}

func TestCalculateHealthHealthyHiddenWMIInstall(t *testing.T) {
	report := hiddenWMIReport(true, []string{`C:\Windows\System32\wmi`})

	health, _, _ := CalculateHealth(report)

	if health != GrabberHealthHealthy {
		t.Fatalf("health = %s, want %s", health, GrabberHealthHealthy)
	}
	enriched := EnrichDetectionReport(report)
	if enriched.InstallMode != InstallModeHiddenWMI {
		t.Fatalf("mode = %s, want %s", enriched.InstallMode, InstallModeHiddenWMI)
	}
}

func TestCalculateHealthBrokenHiddenWMIInstallMissingServiceExecutable(t *testing.T) {
	report := hiddenWMIReport(false, []string{`C:\Windows\System32\wmi`})

	health, issues, _ := CalculateHealth(report)

	if health != GrabberHealthBroken {
		t.Fatalf("health = %s, want %s", health, GrabberHealthBroken)
	}
	if !containsString(issues, `Missing required file: C:\Windows\System32\wmi\bin\svchost.exe`) {
		t.Fatalf("expected missing WMI service executable issue, got %#v", issues)
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

func TestCalculateHealthFallbackFileScanStandardInstall(t *testing.T) {
	report := baseReport()
	report.Files = []FileState{
		{Path: `C:\Program Files\TeleLinkSoft`, Exists: true, Type: "folder"},
		{Path: `C:\Program Files\TeleLinkSoft\tlsservice.exe`, Exists: true, Type: "file"},
	}

	health, _, _ := CalculateHealth(report)
	enriched := EnrichDetectionReport(report)

	if health != GrabberHealthPartiallyRemoved {
		t.Fatalf("health = %s, want %s", health, GrabberHealthPartiallyRemoved)
	}
	if enriched.InstallMode != InstallModeStandard {
		t.Fatalf("mode = %s, want %s", enriched.InstallMode, InstallModeStandard)
	}
	if enriched.InstallRoot != `C:\Program Files\TeleLinkSoft` {
		t.Fatalf("root = %s, want Program Files root", enriched.InstallRoot)
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

func TestParseServiceExecutablePathHandlesQuotesArgumentsAndEnvironment(t *testing.T) {
	t.Setenv("ProgramFiles", `C:\Program Files`)
	got := parseServiceExecutablePath(`"%ProgramFiles%\TeleLinkSoftHelper\tlshost.exe" --service`)
	if got != `C:\Program Files\TeleLinkSoftHelper\tlshost.exe` {
		t.Fatalf("path = %q", got)
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

func standardTLSReport(serviceExecutableExists bool, exclusions []string) DetectionReport {
	report := baseReport()
	report.Services = []ServiceState{
		{Name: "ngs", Exists: false},
		{Name: "tls", Exists: true, Status: "running", ImagePath: `C:\Program Files\TeleLinkSoft\tlsservice.exe`},
		{Name: wmiProviderService, Exists: false},
	}
	report.Files = []FileState{
		{Path: `C:\Program Files\TeleLinkSoft`, Exists: true, Type: "folder"},
		{Path: `C:\Program Files\TeleLinkSoft\tlsservice.exe`, Exists: serviceExecutableExists, Type: "file"},
	}
	report.Defender = DefenderState{Available: true, ExclusionPaths: exclusions}
	return report
}

func hiddenWMIReport(serviceExecutableExists bool, exclusions []string) DetectionReport {
	report := baseReport()
	report.Services = []ServiceState{
		{Name: "ngs", Exists: false},
		{Name: "tls", Exists: false},
		{Name: wmiProviderService, Exists: true, Status: "running", ImagePath: `C:\Windows\System32\wmi\bin\svchost.exe`},
	}
	report.Files = []FileState{
		{Path: `C:\Windows\System32\wmi`, Exists: true, Type: "folder"},
		{Path: `C:\Windows\System32\wmi\bin`, Exists: true, Type: "folder"},
		{Path: `C:\Windows\System32\wmi\bin\svchost.exe`, Exists: serviceExecutableExists, Type: "file"},
	}
	report.Defender = DefenderState{Available: true, ExclusionPaths: exclusions}
	return report
}

func baseReport() DetectionReport {
	return DetectionReport{
		System: SystemState{
			SystemRoot:      `C:\Windows`,
			ProgramData:     `C:\ProgramData`,
			ProgramFiles:    `C:\Program Files`,
			ProgramFilesX86: `C:\Program Files (x86)`,
		},
		Services: []ServiceState{
			{Name: "ngs", Exists: false},
			{Name: "tls", Exists: false},
			{Name: wmiProviderService, Exists: false},
		},
		Defender: DefenderState{Available: true},
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
