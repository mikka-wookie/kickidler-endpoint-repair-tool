package detector

import (
	"strings"
	"testing"
)

func TestKnownProcessNamesExcludeHiddenWMINames(t *testing.T) {
	for _, name := range knownProcessNames() {
		switch strings.ToLower(name) {
		case "svchost.exe", "wmiprvse.exe", "runtimebroker.exe":
			t.Fatalf("hidden WMI process name %q must only match by exact executable path", name)
		}
	}
}

func TestClassifyProcessTrust(t *testing.T) {
	system := testProcessSystem(t)

	tests := []struct {
		name          string
		process       RawProcessInfo
		wantRelated   bool
		wantTrust     string
		wantTerminate bool
		wantMode      string
	}{
		{
			name: "grabber standard path trusted",
			process: RawProcessInfo{ProcessID: 100, Name: "grabber.exe",
				ExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber.exe`},
			wantRelated: true, wantTrust: ProcessTrustNameAndPathMatch, wantTerminate: true, wantMode: string(InstallModeStandard),
		},
		{
			name: "grabber unexpected path mismatch",
			process: RawProcessInfo{ProcessID: 101, Name: "grabber.exe",
				ExecutablePath: `C:\Temp\grabber.exe`},
			wantRelated: true, wantTrust: ProcessTrustPathMismatch, wantTerminate: false,
		},
		{
			name:        "grabber unavailable path",
			process:     RawProcessInfo{ProcessID: 102, Name: "grabber.exe"},
			wantRelated: true, wantTrust: ProcessTrustPathUnavailable, wantTerminate: false,
		},
		{
			name: "x86 normal path trusted",
			process: RawProcessInfo{ProcessID: 103, Name: "ngsAgent.exe",
				ExecutablePath: `C:\Program Files (x86)\TeleLinkSoft\ngsAgent.exe`},
			wantRelated: true, wantTrust: ProcessTrustNameAndPathMatch, wantTerminate: true, wantMode: string(InstallModeStandard),
		},
		{
			name: "helper path trusted",
			process: RawProcessInfo{ProcessID: 104, Name: "tlshost.exe",
				ExecutablePath: `C:\Program Files\TeleLinkSoftHelper\tlshost.exe`},
			wantRelated: true, wantTrust: ProcessTrustNameAndPathMatch, wantTerminate: true, wantMode: string(InstallModeHelper),
		},
		{
			name: "hidden svchost exact trusted",
			process: RawProcessInfo{ProcessID: 105, Name: "svchost.exe",
				ExecutablePath: `C:\Windows\System32\wmi\bin\svchost.exe`},
			wantRelated: true, wantTrust: ProcessTrustHiddenWMIExactPath, wantTerminate: true, wantMode: string(InstallModeHiddenWMI),
		},
		{
			name: "normal svchost not grabber",
			process: RawProcessInfo{ProcessID: 106, Name: "svchost.exe",
				ExecutablePath: `C:\Windows\System32\svchost.exe`},
			wantRelated: false, wantTrust: ProcessTrustPathMismatch, wantTerminate: false,
		},
		{
			name: "hidden WmiPrvSE exact trusted",
			process: RawProcessInfo{ProcessID: 107, Name: "WmiPrvSE.exe",
				ExecutablePath: `C:\Windows\System32\wmi\bin\WmiPrvSE.exe`},
			wantRelated: true, wantTrust: ProcessTrustHiddenWMIExactPath, wantTerminate: true,
		},
		{
			name: "normal WmiPrvSE skipped",
			process: RawProcessInfo{ProcessID: 108, Name: "WmiPrvSE.exe",
				ExecutablePath: `C:\Windows\System32\WmiPrvSE.exe`},
			wantRelated: false, wantTrust: ProcessTrustPathMismatch, wantTerminate: false,
		},
		{
			name: "hidden RuntimeBroker exact trusted",
			process: RawProcessInfo{ProcessID: 109, Name: "RuntimeBroker.exe",
				ExecutablePath: `C:\Windows\System32\wmi\bin\RuntimeBroker.exe`},
			wantRelated: true, wantTrust: ProcessTrustHiddenWMIExactPath, wantTerminate: true,
		},
		{
			name: "normal RuntimeBroker skipped",
			process: RawProcessInfo{ProcessID: 110, Name: "RuntimeBroker.exe",
				ExecutablePath: `C:\Windows\System32\RuntimeBroker.exe`},
			wantRelated: false, wantTrust: ProcessTrustPathMismatch, wantTerminate: false,
		},
		{
			name: "sibling prefix not trusted",
			process: RawProcessInfo{ProcessID: 111, Name: "grabber.exe",
				ExecutablePath: `C:\Program Files\TeleLinkSoft2\grabber.exe`},
			wantRelated: true, wantTrust: ProcessTrustPathMismatch, wantTerminate: false,
		},
		{
			name: "case insensitive trusted",
			process: RawProcessInfo{ProcessID: 112, Name: "GRABBER.EXE",
				ExecutablePath: `c:\program files\telelinksoft\BIN\GRABBER.EXE`},
			wantRelated: true, wantTrust: ProcessTrustNameAndPathMatch, wantTerminate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyProcess(tt.process, MatchOptions{System: system})
			if got.GrabberRelated != tt.wantRelated {
				t.Fatalf("GrabberRelated = %t, want %t: %#v", got.GrabberRelated, tt.wantRelated, got)
			}
			if got.TrustLevel != tt.wantTrust {
				t.Fatalf("TrustLevel = %s, want %s: %#v", got.TrustLevel, tt.wantTrust, got)
			}
			if got.CanTerminate != tt.wantTerminate {
				t.Fatalf("CanTerminate = %t, want %t: %#v", got.CanTerminate, tt.wantTerminate, got)
			}
			if tt.wantMode != "" && got.InstallMode != tt.wantMode {
				t.Fatalf("InstallMode = %s, want %s: %#v", got.InstallMode, tt.wantMode, got)
			}
		})
	}
}

func TestProcessPathNormalization(t *testing.T) {
	system := testProcessSystem(t)
	tests := []struct {
		name string
		path string
		want string
	}{
		{"systemroot expands", `%SystemRoot%\System32\wmi\bin\svchost.exe`, `C:\Windows\System32\wmi\bin\svchost.exe`},
		{"long path prefix", `\\?\C:\Windows\System32\wmi\bin\svchost.exe`, `C:\Windows\System32\wmi\bin\svchost.exe`},
		{"nt object prefix", `\??\C:\Windows\System32\wmi\bin\svchost.exe`, `C:\Windows\System32\wmi\bin\svchost.exe`},
		{"forward slashes", `C:/Windows/System32/wmi/bin/svchost.exe`, `C:\Windows\System32\wmi\bin\svchost.exe`},
		{"trailing slash", `C:\Windows\System32\wmi\bin\`, `C:\Windows\System32\wmi\bin`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeWindowsPath(tt.path); got != tt.want {
				t.Fatalf("NormalizeWindowsPath() = %q, want %q", got, tt.want)
			}
		})
	}

	got := ClassifyProcess(RawProcessInfo{Name: "svchost.exe", ExecutablePath: `%SystemRoot%\System32\wmi\bin\svchost.exe`}, MatchOptions{System: system})
	if got.TrustLevel != ProcessTrustHiddenWMIExactPath || !got.CanTerminate {
		t.Fatalf("expanded hidden WMI path classification = %#v", got)
	}
}

func testProcessSystem(t *testing.T) SystemState {
	t.Helper()
	t.Setenv("SystemRoot", `C:\Windows`)
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)
	t.Setenv("ProgramData", `C:\ProgramData`)
	return SystemState{
		SystemRoot:      `C:\Windows`,
		ProgramFiles:    `C:\Program Files`,
		ProgramFilesX86: `C:\Program Files (x86)`,
		ProgramData:     `C:\ProgramData`,
	}
}
