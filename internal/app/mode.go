package app

type RunMode string

const (
	RunModeInteractive RunMode = "interactive"
	RunModeCLI         RunMode = "cli"
	RunModeQuiet       RunMode = "quiet"
)
