package cliadapter

import (
	"encoding/json"
	"io"

	"kigrepair/internal/app"
)

type GlobalOptions struct {
	Output         string
	ConfigPath     string
	Profile        string
	Quiet          bool
	NonInteractive bool
	Force          bool
	JSONOutput     bool
	Yes            bool
	DryRun         bool
	LogLevel       string
}

func CommonRequest(opts GlobalOptions) app.CommonRequest {
	return app.CommonRequest{
		ConfigPath:     opts.ConfigPath,
		Profile:        opts.Profile,
		OutputDir:      opts.Output,
		JSONOutput:     opts.JSONOutput,
		Quiet:          opts.Quiet,
		NonInteractive: opts.NonInteractive,
		Force:          opts.Force,
		Yes:            opts.Yes,
		DryRun:         opts.DryRun,
		LogLevel:       opts.LogLevel,
	}
}

func RepairRequest(opts GlobalOptions, installerPath string, invite string) app.RepairRequest {
	return app.RepairRequest{
		CommonRequest: CommonRequest(opts),
		InstallerRequestFields: app.InstallerRequestFields{
			InstallerPath: installerPath,
			InviteValue:   invite,
		},
	}
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func ExitError(response *app.WorkflowResponse) error {
	if response == nil || response.Meta.ExitCode == app.ExitSuccess {
		return nil
	}
	return app.ExitError{Code: response.Meta.ExitCode}
}
