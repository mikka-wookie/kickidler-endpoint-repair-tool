package services

import (
	"context"
	"strings"
	"time"

	"kigrepair/internal/winapi"
)

func StopService(ctx context.Context, name string, opts StopOptions) ServiceActionResult {
	opts = normalizeStopOptions(opts)
	if !SupportedServiceName(name) {
		return actionResult("stop", name, "failed", "", "", false, "Service name is not supported", "unsupported service name")
	}
	info := QueryService(name, opts.Runner)
	result := ServiceActionResult{Action: "stop", ServiceName: name, BeforeState: info.State, Trusted: IsTrustedService(info, opts.AllowNameOnly)}
	if !info.Exists {
		result.Status = "skipped"
		result.Message = "Service does not exist"
		return result
	}
	if !result.Trusted {
		result.Status = "failed"
		result.Message = "Service safety validation failed"
		result.Errors = append(result.Errors, "service trust level is "+info.TrustLevel)
		if accessDenied(strings.Join(info.Errors, " ")) {
			result.Warnings = append(result.Warnings, "Run as Administrator")
		}
		return result
	}
	if info.State == StateStopped {
		result.Status = "skipped"
		result.AfterState = StateStopped
		result.Message = "Service is already stopped"
		return result
	}

	if opts.Runner == nil {
		if err := winapi.StopServiceNative(ctx, name); err != nil {
			result.Status = "failed"
			result.Message = "Service stop failed"
			result.Errors = append(result.Errors, err.Error())
			if nativeAccessDenied(err) {
				result.Warnings = append(result.Warnings, "Run as Administrator")
			}
			return result
		}
	} else {
		stop := opts.Runner("sc.exe", "stop", name)
		if stop.ExitCode != 0 && !strings.Contains(stop.Output, "1062") {
			result.Status = "failed"
			result.Message = "Service stop failed"
			result.Errors = append(result.Errors, serviceCommandError(stop))
			if accessDenied(stop.Output) {
				result.Warnings = append(result.Warnings, "Run as Administrator")
			}
			return result
		}
	}
	deadline := time.Now().Add(opts.Timeout)
	for {
		current := QueryService(name, opts.Runner)
		result.AfterState = current.State
		if current.State == StateStopped || !current.Exists {
			result.Status = "success"
			result.Message = "Service stopped"
			return result
		}
		if time.Now().After(deadline) {
			result.Status = "failed"
			result.Message = "Timed out waiting for service to stop"
			result.Errors = append(result.Errors, "last observed state: "+current.State)
			return result
		}
		if !sleepContext(ctx, opts.PollInterval) {
			result.Status = "failed"
			result.Message = "Service stop canceled"
			result.Errors = append(result.Errors, ctx.Err().Error())
			return result
		}
	}
}

func DeleteService(ctx context.Context, name string, opts DeleteOptions) ServiceActionResult {
	opts = normalizeDeleteOptions(opts)
	if !SupportedServiceName(name) {
		return actionResult("delete", name, "failed", "", "", false, "Service name is not supported", "unsupported service name")
	}
	info := QueryService(name, opts.Runner)
	result := ServiceActionResult{Action: "delete", ServiceName: name, BeforeState: info.State, Trusted: IsTrustedService(info, opts.AllowNameOnly)}
	if !info.Exists {
		result.Status = "skipped"
		result.Message = "Service does not exist"
		return result
	}
	if !result.Trusted {
		result.Status = "failed"
		result.Message = "Service safety validation failed"
		result.Errors = append(result.Errors, "service trust level is "+info.TrustLevel)
		if accessDenied(strings.Join(info.Errors, " ")) {
			result.Warnings = append(result.Warnings, "Run as Administrator")
		}
		return result
	}
	if info.State == StateRunning && opts.AllowStop {
		stopped := StopService(ctx, name, StopOptions{Runner: opts.Runner, Timeout: DefaultStopTimeout, PollInterval: opts.PollInterval, AllowNameOnly: opts.AllowNameOnly})
		result.Warnings = append(result.Warnings, stopped.Warnings...)
		if stopped.Status == "failed" {
			result.Status = "failed"
			result.Message = "Service delete blocked because stop failed"
			result.Errors = append(result.Errors, stopped.Errors...)
			return result
		}
	}

	if opts.Runner == nil {
		if err := winapi.DeleteServiceNative(ctx, name); err != nil {
			result.Status = "failed"
			result.Message = "Service deletion failed"
			result.Errors = append(result.Errors, err.Error())
			if nativeAccessDenied(err) {
				result.Warnings = append(result.Warnings, "Run as Administrator")
			}
			return result
		}
	} else {
		del := opts.Runner("sc.exe", "delete", name)
		if del.ExitCode != 0 {
			result.Status = "failed"
			result.Message = "Service deletion failed"
			result.Errors = append(result.Errors, serviceCommandError(del))
			if accessDenied(del.Output) {
				result.Warnings = append(result.Warnings, "Run as Administrator")
			}
			return result
		}
	}
	deadline := time.Now().Add(opts.Timeout)
	for {
		current := QueryService(name, opts.Runner)
		result.AfterState = current.State
		if !current.Exists {
			result.Status = "success"
			result.Message = "Service deleted"
			return result
		}
		if time.Now().After(deadline) {
			result.Status = "warning"
			result.Message = "Service delete command succeeded but service is still visible"
			result.Warnings = append(result.Warnings, "last observed state: "+current.State)
			return result
		}
		if !sleepContext(ctx, opts.PollInterval) {
			result.Status = "failed"
			result.Message = "Service delete verification canceled"
			result.Errors = append(result.Errors, ctx.Err().Error())
			return result
		}
	}
}

func normalizeStopOptions(opts StopOptions) StopOptions {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultStopTimeout
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultQueryRetryInterval
	}
	return opts
}

func normalizeDeleteOptions(opts DeleteOptions) DeleteOptions {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultDeleteVerifyTimeout
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultQueryRetryInterval
	}
	return opts
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func serviceCommandError(result CommandResult) string {
	if strings.TrimSpace(result.Output) != "" {
		return strings.TrimSpace(result.Output)
	}
	if result.Err != nil {
		return result.Err.Error()
	}
	return "service command failed"
}

func accessDenied(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "access is denied") || strings.Contains(lower, "access denied") || strings.Contains(lower, "5:")
}

func nativeAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	return accessDenied(err.Error())
}

func actionResult(action string, name string, status string, before string, after string, trusted bool, message string, errText string) ServiceActionResult {
	result := ServiceActionResult{Action: action, ServiceName: name, Status: status, BeforeState: before, AfterState: after, Trusted: trusted, Message: message}
	if errText != "" {
		result.Errors = append(result.Errors, errText)
	}
	return result
}
