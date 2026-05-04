package config

import "runtime"

const AppName = "kigrepair"

var (
	Version    = "0.1.0"
	GitCommit  = "unknown"
	BuildDate  = "unknown"
	TargetOS   = runtime.GOOS
	TargetArch = runtime.GOARCH
)
