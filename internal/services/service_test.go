package services

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseServiceImagePath(t *testing.T) {
	t.Setenv("SystemRoot", `C:\Windows`)
	tests := []struct {
		name     string
		raw      string
		wantPath string
		wantArgs []string
		wantOK   bool
	}{
		{name: "quoted executable with args", raw: `"C:\Program Files\TeleLinkSoft\bin\grabber.exe" --service`, wantPath: `C:\Program Files\TeleLinkSoft\bin\grabber.exe`, wantArgs: []string{"--service"}, wantOK: true},
		{name: "unquoted executable with args under Program Files", raw: `C:\Program Files\TeleLinkSoft\bin\grabber.exe --service`, wantPath: `C:\Program Files\TeleLinkSoft\bin\grabber.exe`, wantArgs: []string{"--service"}, wantOK: true},
		{name: "env var path", raw: `%SystemRoot%\System32\wmi\bin\svchost.exe -k something`, wantPath: `C:\Windows\System32\wmi\bin\svchost.exe`, wantArgs: []string{"-k", "something"}, wantOK: true},
		{name: "long path prefix", raw: `\\?\C:\Windows\System32\wmi\bin\RuntimeBroker.exe`, wantPath: `C:\Windows\System32\wmi\bin\RuntimeBroker.exe`, wantOK: true},
		{name: "nt path prefix", raw: `\??\C:\Windows\System32\wmi\bin\RuntimeBroker.exe`, wantPath: `C:\Windows\System32\wmi\bin\RuntimeBroker.exe`, wantOK: true},
		{name: "empty image path", raw: ``, wantOK: false},
		{name: "malformed quote", raw: `"C:\Program Files\TeleLinkSoft\bin\grabber.exe --service`, wantOK: false},
		{name: "arguments preserved", raw: `C:\Windows\System32\wmi\bin\WmiPrvSE.exe -a -b`, wantPath: `C:\Windows\System32\wmi\bin\WmiPrvSE.exe`, wantArgs: []string{"-a", "-b"}, wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseServiceImagePath(tt.raw)
			if got.Valid != tt.wantOK {
				t.Fatalf("valid = %v, want %v: %#v", got.Valid, tt.wantOK, got)
			}
			if tt.wantPath != "" && got.NormalizedPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", got.NormalizedPath, tt.wantPath)
			}
			if strings.Join(got.Arguments, "|") != strings.Join(tt.wantArgs, "|") {
				t.Fatalf("args = %#v, want %#v", got.Arguments, tt.wantArgs)
			}
		})
	}
}

func TestClassifyServiceTrust(t *testing.T) {
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)
	t.Setenv("SystemRoot", `C:\Windows`)

	tests := []struct {
		name      string
		imagePath string
		wantTrust string
		wantMode  string
	}{
		{name: "standard path trusted", imagePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, wantTrust: TrustTrusted, wantMode: ModeStandard},
		{name: "helper path trusted", imagePath: `C:\Program Files\TeleLinkSoftHelper\tlshost.exe`, wantTrust: TrustTrusted, wantMode: ModeHelper},
		{name: "x86 path trusted", imagePath: `C:\Program Files (x86)\TeleLinkSoft\grabber.exe`, wantTrust: TrustTrusted, wantMode: ModeStandard},
		{name: "wmi exact path trusted", imagePath: `C:\Windows\System32\wmi\bin\svchost.exe`, wantTrust: TrustTrusted, wantMode: ModeHiddenWMI},
		{name: "normal svchost not trusted", imagePath: `C:\Windows\System32\svchost.exe`, wantTrust: TrustPathMismatch, wantMode: ""},
		{name: "sibling prefix not trusted", imagePath: `C:\Program Files\TeleLinkSoft2\grabber.exe`, wantTrust: TrustPathMismatch, wantMode: ""},
		{name: "unexpected path mismatch", imagePath: `C:\Unexpected\service.exe`, wantTrust: TrustPathMismatch, wantMode: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyService("ngs", true, "running", "", tt.imagePath, "")
			if got.TrustLevel != tt.wantTrust {
				t.Fatalf("trust = %s, want %s: %#v", got.TrustLevel, tt.wantTrust, got)
			}
			if tt.wantMode != "" && got.InstallMode != tt.wantMode {
				t.Fatalf("mode = %s, want %s", got.InstallMode, tt.wantMode)
			}
		})
	}
}

func TestDeriveInstallRootAndMode(t *testing.T) {
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)
	t.Setenv("SystemRoot", `C:\Windows`)

	tests := []struct {
		name     string
		path     string
		wantRoot string
		wantMode string
		trusted  bool
	}{
		{name: "Program Files bin path", path: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, wantRoot: `C:\Program Files\TeleLinkSoft`, wantMode: ModeStandard, trusted: true},
		{name: "helper path", path: `C:\Program Files\TeleLinkSoftHelper\tlshost.exe`, wantRoot: `C:\Program Files\TeleLinkSoftHelper`, wantMode: ModeHelper, trusted: true},
		{name: "WMI bin path", path: `C:\Windows\System32\wmi\bin\RuntimeBroker.exe`, wantRoot: `C:\Windows\System32\wmi`, wantMode: ModeHiddenWMI, trusted: true},
		{name: "unknown path", path: `C:\Other\service.exe`, wantMode: ModeUnknown, trusted: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, mode, trusted := DeriveInstallRootAndMode(tt.path)
			if trusted != tt.trusted || mode != tt.wantMode || (tt.wantRoot != "" && root != tt.wantRoot) {
				t.Fatalf("root=%q mode=%q trusted=%v", root, mode, trusted)
			}
		})
	}
}

func TestServiceActionSafety(t *testing.T) {
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("SystemRoot", `C:\Windows`)

	t.Run("unsupported service cannot be stopped", func(t *testing.T) {
		got := StopService(context.Background(), "Spooler", StopOptions{Runner: newFakeServiceRunner("Spooler", `C:\Windows\System32\spoolsv.exe`, StateRunning).run})
		if got.Status != "failed" {
			t.Fatalf("status = %s, want failed", got.Status)
		}
	})

	t.Run("path mismatch service cannot be stopped", func(t *testing.T) {
		got := StopService(context.Background(), "ngs", StopOptions{Runner: newFakeServiceRunner("ngs", `C:\Unexpected\service.exe`, StateRunning).run})
		if got.Status != "failed" {
			t.Fatalf("status = %s, want failed", got.Status)
		}
	})

	t.Run("missing service stop is skipped", func(t *testing.T) {
		got := StopService(context.Background(), "ngs", StopOptions{Runner: func(string, ...string) CommandResult {
			return CommandResult{ExitCode: 1060, Output: "The specified service does not exist as an installed service."}
		}})
		if got.Status != "skipped" {
			t.Fatalf("status = %s, want skipped", got.Status)
		}
	})

	t.Run("already stopped service stop is skipped", func(t *testing.T) {
		got := StopService(context.Background(), "ngs", StopOptions{Runner: newFakeServiceRunner("ngs", `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, StateStopped).run})
		if got.Status != "skipped" {
			t.Fatalf("status = %s, want skipped", got.Status)
		}
	})

	t.Run("stop timeout returns failed", func(t *testing.T) {
		got := StopService(context.Background(), "ngs", StopOptions{Runner: newFakeServiceRunner("ngs", `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, StateRunning).run, Timeout: time.Millisecond, PollInterval: time.Millisecond})
		if got.Status != "failed" || !strings.Contains(got.Message, "Timed out") {
			t.Fatalf("result = %#v", got)
		}
	})

	t.Run("access denied returns admin action", func(t *testing.T) {
		runner := newFakeServiceRunner("ngs", `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, StateRunning)
		runner.stopResult = CommandResult{ExitCode: 5, Output: "Access is denied."}
		got := StopService(context.Background(), "ngs", StopOptions{Runner: runner.run})
		if got.Status != "failed" || len(got.Warnings) == 0 {
			t.Fatalf("result = %#v", got)
		}
	})

	t.Run("delete verifies service absence", func(t *testing.T) {
		runner := newFakeServiceRunner("ngs", `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, StateStopped)
		runner.deleteRemoves = true
		got := DeleteService(context.Background(), "ngs", DeleteOptions{Runner: runner.run, Timeout: time.Second, PollInterval: time.Millisecond})
		if got.Status != "success" {
			t.Fatalf("result = %#v", got)
		}
	})
}

type fakeServiceRunner struct {
	name          string
	imagePath     string
	state         string
	exists        bool
	deleteRemoves bool
	stopResult    CommandResult
}

func newFakeServiceRunner(name, imagePath, state string) *fakeServiceRunner {
	return &fakeServiceRunner{name: name, imagePath: imagePath, state: state, exists: true}
}

func (f *fakeServiceRunner) run(name string, args ...string) CommandResult {
	if len(args) < 2 || args[1] != f.name || !f.exists {
		return CommandResult{ExitCode: 1060, Output: "The specified service does not exist as an installed service."}
	}
	switch args[0] {
	case "query":
		return CommandResult{Output: "STATE              : 4  " + strings.ToUpper(f.state)}
	case "qc":
		return CommandResult{Output: "START_TYPE         : 2   AUTO_START\nBINARY_PATH_NAME   : " + f.imagePath}
	case "stop":
		if f.stopResult.ExitCode != 0 || f.stopResult.Output != "" {
			return f.stopResult
		}
		return CommandResult{}
	case "delete":
		if f.deleteRemoves {
			f.exists = false
		}
		return CommandResult{}
	default:
		return CommandResult{}
	}
}
