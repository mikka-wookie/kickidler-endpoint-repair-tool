package version

import "runtime"

const ToolName = "kigrepair"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	BuiltBy   = "unknown"
)

type Info struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	BuiltBy   string `json:"built_by"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func Get() Info {
	return Info{
		Tool:      ToolName,
		Version:   nonEmpty(Version, "dev"),
		Commit:    nonEmpty(Commit, "unknown"),
		BuildDate: nonEmpty(BuildDate, "unknown"),
		BuiltBy:   nonEmpty(BuiltBy, "unknown"),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

func SummaryText() string {
	info := Get()
	return info.Tool + " " + info.Version + "\n" +
		"Commit: " + info.Commit + "\n" +
		"Build date: " + info.BuildDate + "\n" +
		"Built by: " + info.BuiltBy + "\n" +
		"Go: " + info.GoVersion + "\n" +
		"Platform: " + info.OS + "/" + info.Arch + "\n"
}

func nonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
