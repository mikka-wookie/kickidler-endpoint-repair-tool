package testfixtures

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/config"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
)

const SecretInvite = "VERY_SECRET_INVITE_SHOULD_NOT_APPEAR"

const (
	StandardRoot       = `C:\Program Files\TeleLinkSoft`
	StandardBin        = `C:\Program Files\TeleLinkSoft\bin`
	StandardGrabberExe = `C:\Program Files\TeleLinkSoft\bin\grabber.exe`
	ProgramDataRoot    = `C:\ProgramData\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60`
	WMIRoot            = `C:\Windows\System32\wmi`
	WMIBin             = `C:\Windows\System32\wmi\bin`
	WMIProviderExe     = `C:\Windows\System32\wmi\bin\WmiPrvSE.exe`
)

type Scenario struct {
	Name         string
	Description  string
	IsAdmin      bool
	OS           string
	Architecture string

	Services     []FakeService
	Processes    []FakeProcess
	Files        map[string]FakeFile
	Directories  map[string]FakeDirectory
	RegistryKeys map[string]FakeRegistryKey
	Defender     FakeDefender
	Installer    FakeInstaller

	Expected ScenarioExpected
}

type ScenarioExpected struct {
	Health                   string
	InstallMode              string
	InstallRoot              string
	PrimaryIssueCode         string
	RecommendationCode       string
	RecommendationStatus     string
	PreflightStatus          string
	RepairPlanStatus         string
	CleanupActionsMin        int
	DefenderCovered          bool
	ShouldRequireAdmin       bool
	ShouldRequireInvite      bool
	ShouldRequireInstaller   bool
	NoCleanupTerminateAction bool
}

func BaseSystem() detector.SystemState {
	return detector.SystemState{
		GOOS:            "windows",
		GOARCH:          "amd64",
		Windows:         "10.0.19045",
		Hostname:        "fixture-host",
		Username:        "fixture-user",
		SystemRoot:      `C:\Windows`,
		ProgramFiles:    `C:\Program Files`,
		ProgramFilesX86: `C:\Program Files (x86)`,
		ProgramData:     `C:\ProgramData`,
	}
}

func ScenarioByName(name string) Scenario {
	for _, scenario := range Scenarios() {
		if scenario.Name == name {
			return scenario
		}
	}
	panic("unknown scenario: " + name)
}

func Scenarios() []Scenario {
	return []Scenario{
		healthyStandardInstall(),
		defenderDeletedServiceBinary(),
		notInstalledCleanMachine(),
		partialMSILeftovers(),
		partialFilesLeftover(),
		hiddenWMIHealthy(),
		hiddenWMIServiceBinaryMissing(),
		normalWindowsSvchostNotGrabber(),
		supportedServiceNamePathMismatch(),
		processNameMatchPathMismatch(),
		defenderParentExclusionCoversChild(),
		defenderPrefixFalsePositive(),
		defenderQueryFailed(),
		invalidInstallerBlocksRepair(),
		missingInviteBlocksRepairReadiness(),
		missingAdminBlocksRepairReadiness(),
	}
}

func (s Scenario) DetectionReport() detector.DetectionReport {
	system := BaseSystem()
	if s.OS != "" {
		system.GOOS = s.OS
	}
	if s.Architecture != "" {
		system.GOARCH = s.Architecture
	}
	report := detector.DetectionReport{
		GeneratedAt: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
		IsAdmin:     s.IsAdmin,
		System:      system,
		Services:    servicesFromScenario(s),
		Files:       filesFromScenario(s),
		Processes:   processesFromScenario(s, system),
		Registry:    registryFromScenario(s),
		Defender: detector.DefenderState{
			Available:      !s.Defender.QueryFailed,
			ExclusionPaths: append([]string{}, s.Defender.ExclusionPaths...),
			Error:          s.Defender.Error,
		},
	}
	report = detector.EnrichDetectionReport(report)
	report.Health, report.Issues, report.Recommendations = detector.CalculateHealth(report)
	return report
}

func (s Scenario) PathExists(path string) (bool, error) {
	normalized := detector.NormalizeWindowsPath(path)
	for candidate, file := range s.Files {
		if file.Error != "" && pathsEqual(candidate, normalized) {
			return false, errors.New(file.Error)
		}
		if pathsEqual(candidate, normalized) {
			return file.Exists, nil
		}
	}
	for candidate, dir := range s.Directories {
		if dir.Error != "" && pathsEqual(candidate, normalized) {
			return false, errors.New(dir.Error)
		}
		if pathsEqual(candidate, normalized) {
			return dir.Exists, nil
		}
	}
	return false, nil
}

func (s Scenario) CleanupPaths() []string {
	paths := []string{}
	for path, dir := range s.Directories {
		if dir.Exists {
			paths = append(paths, path)
		}
	}
	return paths
}

func (s Scenario) InstallerResolver(path string) (installer.InstallerResolution, error) {
	fake := s.Installer
	selected := firstNonEmpty(path, fake.Path)
	validation := installer.ValidationResult{
		Status:      installer.ValidationStatusInvalid,
		Path:        selected,
		FileName:    filepath.Base(selected),
		Exists:      fake.Exists,
		IsFile:      fake.Exists,
		Readable:    fake.Exists,
		ExtensionOK: strings.EqualFold(filepath.Ext(selected), ".msi"),
	}
	if fake.Valid {
		validation.Status = installer.ValidationStatusValid
		validation.ExtensionOK = true
		validation.SHA256 = "fixture-sha256"
	}
	if fake.ValidationStatus != "" {
		validation.Status = fake.ValidationStatus
	}
	if validation.Status == installer.ValidationStatusInvalid {
		validation.Errors = append(validation.Errors, firstNonEmpty(fake.Error, "fixture installer is invalid"))
	}
	resolution := installer.InstallerResolution{
		SelectedPath:   selected,
		SelectedSource: installer.InstallerSourceExplicit,
		Candidates: []installer.InstallerCandidate{{
			Path:       selected,
			Name:       filepath.Base(selected),
			Source:     installer.InstallerSourceExplicit,
			Exists:     fake.Exists,
			Validation: validation,
		}},
		OSArchitecture: "amd64",
	}
	if !fake.Exists {
		resolution.SelectedSource = installer.InstallerSourceNotFound
		return resolution, installer.ErrInstallerNotFound
	}
	if !validation.IsUsable() {
		resolution.Error = validation.ErrorSummary()
		return resolution, errors.New(validation.ErrorSummary())
	}
	return resolution, nil
}

func ValidInstaller(path string) FakeInstaller {
	return FakeInstaller{Path: path, Exists: true, Valid: true, ValidationStatus: installer.ValidationStatusValid}
}

func InvalidInstaller(path string) FakeInstaller {
	return FakeInstaller{Path: path, Exists: true, Valid: false, ValidationStatus: installer.ValidationStatusInvalid, Error: "File is an MSI package: .txt"}
}

func healthyStandardInstall() Scenario {
	return Scenario{
		Name:        "healthy_standard_install",
		Description: "Running standard service with existing binary and parent Defender exclusion.",
		IsAdmin:     true,
		Services:    []FakeService{{Name: "ngs", Exists: true, Status: "running", ImagePath: StandardGrabberExe}},
		Processes:   []FakeProcess{{PID: 1001, Name: "grabber.exe", ExecutablePath: StandardGrabberExe}},
		Files:       map[string]FakeFile{StandardGrabberExe: {Exists: true}},
		Directories: map[string]FakeDirectory{StandardRoot: {Exists: true}, StandardBin: {Exists: true}},
		RegistryKeys: map[string]FakeRegistryKey{
			`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + config.MSIProductCode: {Exists: true},
		},
		Defender:  FakeDefender{ExclusionPaths: []string{StandardRoot}},
		Installer: ValidInstaller(`C:\Support\grabberEM.x64.msi`),
		Expected: ScenarioExpected{
			Health:             string(detector.GrabberHealthHealthy),
			InstallMode:        string(detector.InstallModeStandard),
			InstallRoot:        StandardRoot,
			PrimaryIssueCode:   "healthy",
			RecommendationCode: "no_repair_required",
			DefenderCovered:    true,
		},
	}
}

func defenderDeletedServiceBinary() Scenario {
	s := healthyStandardInstall()
	s.Name = "defender_deleted_service_binary"
	s.Files[StandardGrabberExe] = FakeFile{Exists: false}
	s.Defender = FakeDefender{}
	s.Expected.Health = string(detector.GrabberHealthBroken)
	s.Expected.PrimaryIssueCode = "service_binary_missing"
	s.Expected.RecommendationCode = "run_full_repair"
	s.Expected.RepairPlanStatus = "not_ready"
	s.Expected.DefenderCovered = false
	return s
}

func notInstalledCleanMachine() Scenario {
	return Scenario{
		Name:      "not_installed_clean_machine",
		Defender:  FakeDefender{ExclusionPaths: nil},
		Installer: ValidInstaller(`C:\Support\grabberEM.x64.msi`),
		Expected: ScenarioExpected{
			Health:             string(detector.GrabberHealthNotInstalled),
			InstallMode:        string(detector.InstallModeNotInstalled),
			PrimaryIssueCode:   "not_installed",
			RecommendationCode: "install_or_repair_with_invite",
		},
	}
}

func partialMSILeftovers() Scenario {
	return Scenario{
		Name: "partial_msi_leftovers",
		RegistryKeys: map[string]FakeRegistryKey{
			`HKCR\Installer\Products\` + config.MSIPackedCode:                                   {Exists: true},
			`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + config.MSIProductCode: {Exists: true},
		},
		Defender:  FakeDefender{},
		Installer: ValidInstaller(`C:\Support\grabberEM.x64.msi`),
		Expected: ScenarioExpected{
			Health:             string(detector.GrabberHealthPartiallyRemoved),
			PrimaryIssueCode:   "partial_msi_leftovers",
			RecommendationCode: "run_cleanup_dry_run",
			CleanupActionsMin:  2,
		},
	}
}

func partialFilesLeftover() Scenario {
	return Scenario{
		Name:        "partial_files_leftover",
		Directories: map[string]FakeDirectory{StandardRoot: {Exists: true}, ProgramDataRoot: {Exists: true}},
		Files:       map[string]FakeFile{filepath.Join(StandardRoot, "old.dat"): {Exists: true}},
		Defender:    FakeDefender{},
		Installer:   ValidInstaller(`C:\Support\grabberEM.x64.msi`),
		Expected: ScenarioExpected{
			Health:             string(detector.GrabberHealthPartiallyRemoved),
			PrimaryIssueCode:   "partial_files_leftover",
			RecommendationCode: "run_cleanup_dry_run",
			CleanupActionsMin:  1,
		},
	}
}

func hiddenWMIHealthy() Scenario {
	return Scenario{
		Name:      "hidden_wmi_healthy",
		IsAdmin:   true,
		Services:  []FakeService{{Name: "WmiProviderSE", Exists: true, Status: "running", ImagePath: WMIProviderExe}},
		Processes: []FakeProcess{{PID: 2001, Name: "WmiPrvSE.exe", ExecutablePath: WMIProviderExe}, {PID: 4, Name: "svchost.exe", ExecutablePath: `C:\Windows\System32\svchost.exe`}},
		Files: map[string]FakeFile{
			filepath.Join(WMIBin, "svchost.exe"):       {Exists: true},
			filepath.Join(WMIBin, "WmiPrvSE.exe"):      {Exists: true},
			filepath.Join(WMIBin, "RuntimeBroker.exe"): {Exists: true},
		},
		Directories: map[string]FakeDirectory{WMIRoot: {Exists: true}, WMIBin: {Exists: true}},
		Defender:    FakeDefender{ExclusionPaths: []string{WMIRoot}},
		Installer:   ValidInstaller(`C:\Support\grabberEM.x64.msi`),
		Expected: ScenarioExpected{
			Health:             string(detector.GrabberHealthHealthy),
			InstallMode:        string(detector.InstallModeHiddenWMI),
			InstallRoot:        WMIRoot,
			PrimaryIssueCode:   "healthy",
			RecommendationCode: "no_repair_required",
			DefenderCovered:    true,
		},
	}
}

func hiddenWMIServiceBinaryMissing() Scenario {
	s := hiddenWMIHealthy()
	s.Name = "hidden_wmi_service_binary_missing"
	s.Files[WMIProviderExe] = FakeFile{Exists: false}
	s.Expected.Health = string(detector.GrabberHealthBroken)
	s.Expected.PrimaryIssueCode = "wmi_hidden_mode_inconsistent"
	s.Expected.RecommendationCode = "run_full_repair"
	return s
}

func normalWindowsSvchostNotGrabber() Scenario {
	return Scenario{
		Name:      "normal_windows_svchost_not_grabber",
		Processes: []FakeProcess{{PID: 4, Name: "svchost.exe", ExecutablePath: `C:\Windows\System32\svchost.exe`}},
		Defender:  FakeDefender{},
		Expected: ScenarioExpected{
			Health:                   string(detector.GrabberHealthNotInstalled),
			InstallMode:              string(detector.InstallModeNotInstalled),
			NoCleanupTerminateAction: true,
		},
	}
}

func supportedServiceNamePathMismatch() Scenario {
	return Scenario{
		Name:      "supported_service_name_path_mismatch",
		Services:  []FakeService{{Name: "ngs", Exists: true, Status: "running", ImagePath: `C:\Unexpected\ngs.exe`}},
		Defender:  FakeDefender{},
		Installer: ValidInstaller(`C:\Support\grabberEM.x64.msi`),
		Expected: ScenarioExpected{
			Health:             string(detector.GrabberHealthBroken),
			InstallMode:        string(detector.InstallModeUnknown),
			PrimaryIssueCode:   "unknown_install_state",
			RecommendationCode: "collect_support_bundle",
		},
	}
}

func processNameMatchPathMismatch() Scenario {
	return Scenario{
		Name:      "process_name_match_path_mismatch",
		Processes: []FakeProcess{{PID: 3001, Name: "grabber.exe", ExecutablePath: `C:\Temp\grabber.exe`}},
		Defender:  FakeDefender{},
		Expected: ScenarioExpected{
			Health:                   string(detector.GrabberHealthNotInstalled),
			InstallMode:              string(detector.InstallModeNotInstalled),
			NoCleanupTerminateAction: true,
		},
	}
}

func defenderParentExclusionCoversChild() Scenario {
	return Scenario{
		Name:     "defender_parent_exclusion_covers_child",
		Defender: FakeDefender{RequiredPaths: []string{StandardBin}, ExclusionPaths: []string{StandardRoot}},
		Expected: ScenarioExpected{DefenderCovered: true},
	}
}

func defenderPrefixFalsePositive() Scenario {
	return Scenario{
		Name:     "defender_prefix_false_positive",
		Defender: FakeDefender{RequiredPaths: []string{StandardRoot}, ExclusionPaths: []string{StandardRoot + "2"}},
		Expected: ScenarioExpected{DefenderCovered: false},
	}
}

func defenderQueryFailed() Scenario {
	s := healthyStandardInstall()
	s.Name = "defender_query_failed"
	s.Defender = FakeDefender{QueryFailed: true, Error: "Get-MpPreference failed"}
	s.Expected.PrimaryIssueCode = "defender_status_unavailable"
	s.Expected.RecommendationCode = "collect_support_bundle"
	return s
}

func invalidInstallerBlocksRepair() Scenario {
	s := defenderDeletedServiceBinary()
	s.Name = "invalid_installer_blocks_repair"
	s.IsAdmin = true
	s.Installer = InvalidInstaller(`C:\Support\not-an-msi.txt`)
	s.Expected.PreflightStatus = "not_ready"
	s.Expected.RepairPlanStatus = "not_ready"
	return s
}

func missingInviteBlocksRepairReadiness() Scenario {
	s := defenderDeletedServiceBinary()
	s.Name = "missing_invite_blocks_repair_readiness"
	s.IsAdmin = true
	s.Installer = ValidInstaller(`C:\Support\grabberEM.x64.msi`)
	s.Expected.PreflightStatus = "not_ready"
	s.Expected.RepairPlanStatus = "not_ready"
	s.Expected.ShouldRequireInvite = true
	return s
}

func missingAdminBlocksRepairReadiness() Scenario {
	s := defenderDeletedServiceBinary()
	s.Name = "missing_admin_blocks_repair_readiness"
	s.IsAdmin = false
	s.Installer = ValidInstaller(`C:\Support\grabberEM.x64.msi`)
	s.Expected.PreflightStatus = "not_ready"
	s.Expected.RepairPlanStatus = "not_ready"
	s.Expected.ShouldRequireAdmin = true
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
