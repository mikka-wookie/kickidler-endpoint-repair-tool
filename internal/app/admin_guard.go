package app

import (
	"errors"
	"fmt"
)

var (
	ErrAdminRequired    = errors.New("administrator rights are required")
	ErrElevationFailed  = errors.New("elevation was cancelled or failed")
	ErrElevationMissing = errors.New("elevation failed; administrator rights are still not available")
)

type AdminChecker interface {
	IsAdmin() bool
}

type ElevationLauncher interface {
	RelaunchElevated(args []string) error
}

type AdminGuardOptions struct {
	CommandName    string
	RequiresAdmin  bool
	NoElevate      bool
	Quiet          bool
	NonInteractive bool
	ElevatedChild  bool
	Args           []string
	AdminChecker   AdminChecker
	Launcher       ElevationLauncher
}

type AdminGuardResult struct {
	Relaunched bool
	ExitCode   int
	Message    string
	Err        error
}

func EnsureAdminOrRelaunch(opts AdminGuardOptions) AdminGuardResult {
	if !opts.RequiresAdmin {
		return AdminGuardResult{}
	}
	if opts.AdminChecker == nil {
		return adminRequiredResult(opts, ErrAdminRequired)
	}
	if opts.AdminChecker.IsAdmin() {
		return AdminGuardResult{}
	}
	if opts.ElevatedChild {
		return AdminGuardResult{
			ExitCode: ExitAdminRequired,
			Message:  "Elevation failed. Administrator rights are still not available.",
			Err:      ErrElevationMissing,
		}
	}
	if opts.NoElevate {
		return adminRequiredResult(opts, ErrAdminRequired)
	}
	if opts.Quiet || opts.NonInteractive {
		return AdminGuardResult{
			ExitCode: ExitAdminRequired,
			Message:  "Administrator rights are required. Re-run from elevated PowerShell or omit --non-interactive to allow UAC prompt.",
			Err:      ErrAdminRequired,
		}
	}
	if opts.Launcher == nil {
		return AdminGuardResult{
			ExitCode: ExitUnexpectedError,
			Message:  "Elevation was cancelled or failed.",
			Err:      ErrElevationFailed,
		}
	}
	if err := opts.Launcher.RelaunchElevated(AppendElevatedChildArg(opts.Args)); err != nil {
		return AdminGuardResult{
			ExitCode: ExitUnexpectedError,
			Message:  "Elevation was cancelled or failed.",
			Err:      fmt.Errorf("%w: %v", ErrElevationFailed, err),
		}
	}
	return AdminGuardResult{Relaunched: true}
}

func adminRequiredResult(opts AdminGuardOptions, err error) AdminGuardResult {
	command := opts.CommandName
	if command == "" {
		command = "this command"
	}
	return AdminGuardResult{
		ExitCode: ExitAdminRequired,
		Message:  fmt.Sprintf("Administrator rights are required for %s.\nRun PowerShell as Administrator or remove --no-elevate to allow UAC prompt.", command),
		Err:      err,
	}
}
