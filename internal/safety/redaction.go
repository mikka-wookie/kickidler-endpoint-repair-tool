package safety

import (
	"regexp"
	"strings"
)

const RedactedValue = "<REDACTED>"

var redactionPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{
		pattern:     regexp.MustCompile(`(?i)("(?:invite|access_token|refresh_token|token|password|secret)"\s*:\s*")[^"]*(")`),
		replacement: `${1}` + RedactedValue + `${2}`,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(\b(?:invite|access_token|refresh_token|token|password|secret)\s*=\s*)[^\s,"';]+`),
		replacement: `${1}` + RedactedValue,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(\b(?:invite|access_token|refresh_token|token|password|secret)\s*:\s*)[^\s,"';]+`),
		replacement: `${1}` + RedactedValue,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(Authorization\s*:\s*(?:Bearer|Basic)\s+)[^\s\r\n]+`),
		replacement: `${1}` + RedactedValue,
	},
}

func RedactString(value string) string {
	redacted := value
	for _, item := range redactionPatterns {
		redacted = item.pattern.ReplaceAllString(redacted, item.replacement)
	}
	return redacted
}

func RedactStringWithSecrets(value string, secrets ...string) string {
	redacted := value
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, RedactedValue)
	}
	return RedactString(redacted)
}

func RedactBytes(value []byte) []byte {
	return []byte(RedactString(string(value)))
}
