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
