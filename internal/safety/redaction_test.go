package safety

import (
	"strings"
	"testing"
)

func TestRedactStringRedactsSensitiveValues(t *testing.T) {
	tests := []string{
		"invite=abc123",
		"invite: abc123",
		`"invite":"abc123"`,
		"token=abc123",
		"access_token=abc123",
		"refresh_token=abc123",
		"password=abc123",
		"secret=abc123",
		"Authorization: Bearer abc123",
		"Authorization: Basic abc123",
	}
	for _, input := range tests {
		got := RedactString(input)
		if strings.Contains(got, "abc123") {
			t.Fatalf("RedactString(%q) exposed secret: %q", input, got)
		}
		if !strings.Contains(got, RedactedValue) {
			t.Fatalf("RedactString(%q) = %q, want redaction marker", input, got)
		}
	}
}

func TestRedactStringDoesNotRedactExpectedOperationalValues(t *testing.T) {
	input := strings.Join([]string{
		`product={EB1FBC37-0B97-4CF5-A329-CF28BA653748}`,
		`packed=73CBF1BE79B05FC43A92FC82AB567384`,
		`path=C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		`service=ngs`,
		`health=healthy`,
		`install_mode=standard`,
	}, "\n")
	got := RedactString(input)
	for _, want := range []string{
		"{EB1FBC37-0B97-4CF5-A329-CF28BA653748}",
		"73CBF1BE79B05FC43A92FC82AB567384",
		`C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		"ngs",
		"healthy",
		"standard",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RedactString removed %q from %q", want, got)
		}
	}
}
