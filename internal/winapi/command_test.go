package winapi

import (
	"os"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/failures"
)

func TestRunCommandTimeoutReturnsStructuredFailure(t *testing.T) {
	result := RunCommand(CommandOptions{
		Name:    os.Args[0],
		Args:    helperArgs("timeout"),
		Env:     helperEnv(),
		Timeout: 10 * time.Millisecond,
	})
	if !result.TimedOut {
		t.Fatalf("TimedOut = false, want true")
	}
	if result.Category != failures.FailureExternalTimeout {
		t.Fatalf("Category = %q, want %q", result.Category, failures.FailureExternalTimeout)
	}
}

func TestRunCommandNonZeroExitCode(t *testing.T) {
	result := RunCommand(CommandOptions{
		Name:    os.Args[0],
		Args:    helperArgs("nonzero"),
		Env:     helperEnv(),
		Timeout: time.Second,
	})
	if result.ExitCode != 7 {
		t.Fatalf("ExitCode = %d, want 7", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "bad") {
		t.Fatalf("Stderr = %q, want bad", result.Stderr)
	}
}

func TestRunCommandRedactsOutputAndInviteArgument(t *testing.T) {
	const invite = "raw-secret-invite"
	result := RunCommand(CommandOptions{
		Name:          os.Args[0],
		Args:          append(helperArgs("secret"), "invite="+invite),
		Env:           helperEnv(),
		Timeout:       time.Second,
		SensitiveArgs: []string{invite},
	})
	joined := strings.Join([]string{strings.Join(result.Args, " "), result.CommandLine, result.Stdout, result.Stderr, result.Error}, "\n")
	if strings.Contains(joined, invite) || strings.Contains(joined, "token=abc") {
		t.Fatalf("result exposed sensitive value: %#v", result)
	}
	if !strings.Contains(result.CommandLine, "invite=<REDACTED>") {
		t.Fatalf("CommandLine = %q, want redacted invite", result.CommandLine)
	}
}

func TestRunCommandEnvironmentDoesNotLeakSensitiveArg(t *testing.T) {
	result := RunCommand(CommandOptions{
		Name:          os.Args[0],
		Args:          helperArgs("env-secret"),
		Timeout:       time.Second,
		Env:           append(helperEnv(), "KIGREPAIR_SECRET=abc123"),
		SensitiveArgs: []string{"abc123"},
	})
	if strings.Contains(result.Stdout, "abc123") {
		t.Fatalf("Stdout exposed secret: %q", result.Stdout)
	}
}

func TestCommandHelperProcess(t *testing.T) {
	if os.Getenv("KIGREPAIR_COMMAND_HELPER") != "1" {
		return
	}
	mode := ""
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
			break
		}
	}
	switch mode {
	case "timeout":
		time.Sleep(2 * time.Second)
	case "nonzero":
		_, _ = os.Stderr.WriteString("bad\n")
		os.Exit(7)
	case "secret":
		_, _ = os.Stdout.WriteString("invite=raw-secret-invite\n")
		_, _ = os.Stderr.WriteString("token=abc\n")
	case "env-secret":
		_, _ = os.Stdout.WriteString(os.Getenv("KIGREPAIR_SECRET") + "\n")
	}
	os.Exit(0)
}

func helperArgs(mode string) []string {
	return []string{"-test.run=TestCommandHelperProcess", "--", mode}
}

func helperEnv() []string {
	return []string{"KIGREPAIR_COMMAND_HELPER=1"}
}
