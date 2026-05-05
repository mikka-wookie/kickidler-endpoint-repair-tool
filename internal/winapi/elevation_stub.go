//go:build !windows

package winapi

import (
	"fmt"
	"strings"
)

func IsAdmin() bool {
	return false
}

func RelaunchElevated(args []string) error {
	return fmt.Errorf("%w: elevation is Windows-only", ErrUnsupportedPlatform)
}

func QuoteWindowsArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, QuoteWindowsArg(arg))
	}
	return strings.Join(quoted, " ")
}

func QuoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			backslashes++
		case '"':
			b.WriteString(strings.Repeat(`\`, backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
		default:
			if backslashes > 0 {
				b.WriteString(strings.Repeat(`\`, backslashes))
				backslashes = 0
			}
			b.WriteRune(r)
		}
	}
	if backslashes > 0 {
		b.WriteString(strings.Repeat(`\`, backslashes*2))
	}
	b.WriteByte('"')
	return b.String()
}

func appendElevatedChildArg(args []string) []string {
	for _, arg := range args {
		if arg == "--elevated-child" {
			return append([]string(nil), args...)
		}
	}
	childArgs := append([]string(nil), args...)
	return append(childArgs, "--elevated-child")
}
