package main

import "testing"

func TestQuoteCommandLineArgs(t *testing.T) {
	got := quoteCommandLineArgs([]string{
		"check",
		"--output",
		`C:\ProgramData\kigrepair\Reports\with spaces`,
		`--name=value"quoted"`,
		`trailing\`,
	})
	want := `check --output "C:\ProgramData\kigrepair\Reports\with spaces" "--name=value\"quoted\"" trailing\`
	if got != want {
		t.Fatalf("quoteCommandLineArgs() = %q, want %q", got, want)
	}
}
