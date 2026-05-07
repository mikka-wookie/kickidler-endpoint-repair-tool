package winapi

import "testing"

func TestQuoteWindowsArgs(t *testing.T) {
	got := QuoteWindowsArgs([]string{
		"check",
		"--output",
		`C:\ProgramData\kigrepair\Reports\with spaces`,
		`--name=value"quoted"`,
		`trailing\`,
	})
	want := `check --output "C:\ProgramData\kigrepair\Reports\with spaces" "--name=value\"quoted\"" trailing\`
	if got != want {
		t.Fatalf("QuoteWindowsArgs() = %q, want %q", got, want)
	}
}

func TestQuoteWindowsArgsPreservesInstallerPathWithSpaces(t *testing.T) {
	got := QuoteWindowsArgs([]string{
		"install",
		"--installer",
		`C:\Installers\Kickidler Grabber\grabber.msi`,
		"--invite",
		"TEST",
	})
	want := `install --installer "C:\Installers\Kickidler Grabber\grabber.msi" --invite TEST`
	if got != want {
		t.Fatalf("QuoteWindowsArgs() = %q, want %q", got, want)
	}
}

func TestAppendElevatedChildArgDoesNotDuplicate(t *testing.T) {
	got := appendElevatedChildArg([]string{"--elevated-child", "--profile", "diagnostic"})
	if len(got) != 3 {
		t.Fatalf("args = %#v", got)
	}
}

func TestGUIRelaunchArgsShapeDoesNotRequireInvite(t *testing.T) {
	args := appendElevatedChildArg([]string{"--no-auto-workflow", "--profile", "diagnostic"})
	quoted := QuoteWindowsArgs(args)
	if quoted != "--no-auto-workflow --profile diagnostic --elevated-child" {
		t.Fatalf("quoted args = %q", quoted)
	}
}
