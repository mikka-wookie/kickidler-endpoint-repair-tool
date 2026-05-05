package winapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"kigrepair/internal/failures"
	"kigrepair/internal/safety"
)

const (
	DefaultPowerShellTimeout       = 30 * time.Second
	DefaultServiceQueryTimeout     = 10 * time.Second
	DefaultServiceStopTimeout      = 30 * time.Second
	DefaultServiceDeleteTimeout    = 10 * time.Second
	DefaultProcessQueryTimeout     = 15 * time.Second
	DefaultProcessTerminateTimeout = 10 * time.Second
	DefaultRegistryQueryTimeout    = 10 * time.Second
	DefaultMSITimeout              = 10 * time.Minute
	DefaultSupportBundleTimeout    = 2 * time.Minute
	DefaultReportCleanupTimeout    = 30 * time.Second
)

type CommandOptions struct {
	Name          string
	Args          []string
	Timeout       time.Duration
	RedactArgs    bool
	SensitiveArgs []string
	WorkingDir    string
	Env           []string
	Category      string
}

type CommandResult struct {
	Name        string   `json:"name"`
	Args        []string `json:"args,omitempty"`
	CommandLine string   `json:"command_line,omitempty"`
	ExitCode    int      `json:"exit_code"`
	Stdout      string   `json:"stdout,omitempty"`
	Stderr      string   `json:"stderr,omitempty"`
	TimedOut    bool     `json:"timed_out"`
	DurationMS  int64    `json:"duration_ms"`
	Error       string   `json:"error,omitempty"`
	Category    string   `json:"category,omitempty"`
}

func RunCommand(opts CommandOptions) CommandResult {
	started := time.Now()
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	category := opts.Category
	if category == "" {
		category = failures.FailureExternalCommand
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.Name, opts.Args...)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	result := CommandResult{
		Name:        opts.Name,
		Args:        redactArgs(opts.Args, opts),
		CommandLine: CommandLine(opts.Name, opts.Args, opts),
		ExitCode:    0,
		Stdout:      redactText(stdout.String(), opts),
		Stderr:      redactText(stderr.String(), opts),
		TimedOut:    ctx.Err() == context.DeadlineExceeded,
		DurationMS:  time.Since(started).Milliseconds(),
		Category:    category,
	}
	if result.TimedOut {
		result.ExitCode = -1
		result.Category = failures.FailureExternalTimeout
		result.Error = fmt.Sprintf("command timed out after %s", timeout)
		return result
	}
	if err != nil {
		result.ExitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = safety.RedactString(err.Error())
	}
	result.Stdout = strings.TrimSpace(result.Stdout)
	result.Stderr = strings.TrimSpace(result.Stderr)
	return result
}

func CommandLine(name string, args []string, opts CommandOptions) string {
	all := append([]string{name}, redactArgs(args, opts)...)
	return safety.RedactString(QuoteWindowsArgs(all))
}

func redactArgs(args []string, opts CommandOptions) []string {
	redacted := make([]string, len(args))
	copy(redacted, args)
	if opts.RedactArgs {
		for i := range redacted {
			redacted[i] = safety.RedactedValue
		}
	}
	for i, arg := range redacted {
		if isSensitiveArg(arg, opts.SensitiveArgs) {
			redacted[i] = redactSensitiveArg(arg)
		}
	}
	return redacted
}

func redactText(value string, opts CommandOptions) string {
	redacted := safety.RedactString(value)
	for _, sensitive := range opts.SensitiveArgs {
		sensitive = strings.TrimSpace(sensitive)
		if sensitive == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, sensitive, safety.RedactedValue)
	}
	return redacted
}

func isSensitiveArg(arg string, sensitive []string) bool {
	lower := strings.ToLower(strings.TrimSpace(arg))
	if strings.HasPrefix(lower, "invite=") ||
		strings.HasPrefix(lower, "token=") ||
		strings.HasPrefix(lower, "password=") ||
		strings.HasPrefix(lower, "secret=") {
		return true
	}
	for _, value := range sensitive {
		value = strings.TrimSpace(value)
		if value != "" && strings.Contains(arg, value) {
			return true
		}
	}
	return false
}

func redactSensitiveArg(arg string) string {
	if idx := strings.Index(arg, "="); idx >= 0 {
		return arg[:idx+1] + safety.RedactedValue
	}
	return safety.RedactedValue
}

func (r CommandResult) CombinedOutput() string {
	switch {
	case r.Stdout != "" && r.Stderr != "":
		return r.Stdout + "\n" + r.Stderr
	case r.Stdout != "":
		return r.Stdout
	default:
		return r.Stderr
	}
}
