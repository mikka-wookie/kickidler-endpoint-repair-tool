package config

var Services = []string{
	"ngs",
	"tls",
	"WmiProviderSE",
}

var CleanupPaths = []string{
	`%ProgramFiles%\TeleLinkSoft`,
	`%ProgramFiles%\TeleLinkSoftHelper`,
	`%ProgramFiles(x86)%\TeleLinkSoft`,
	`%ProgramFiles(x86)%\TeleLinkSoftHelper`,
	`%ProgramData%\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60`,
	`%SystemRoot%\System32\wmi`,
}

var DefenderExclusionPaths = []string{
	`%ProgramFiles%\TeleLinkSoft`,
	`%ProgramFiles%\TeleLinkSoftHelper`,
	`%ProgramFiles(x86)%\TeleLinkSoft`,
	`%ProgramFiles(x86)%\TeleLinkSoftHelper`,
	`%SystemRoot%\System32\wmi`,
}

var KnownProcessNames = []string{
	"grabber.exe",
	"grabberAgent.exe",
	"grabberSubAgent.exe",
	"grabberSubagent.exe",
	"grabber2.exe",
	"ngsAgent.exe",
	"ngsSubAgent.exe",
	"ngsSubagent.exe",
	"tlshost.exe",
	"tlsservice.exe",
	"tlssubservice.exe",
}

var KnownWMIExecutablePaths = []string{
	`%SystemRoot%\System32\wmi\bin\svchost.exe`,
	`%SystemRoot%\System32\wmi\bin\WmiPrvSE.exe`,
	`%SystemRoot%\System32\wmi\bin\RuntimeBroker.exe`,
}

const (
	MSIProductCode = "{EB1FBC37-0B97-4CF5-A329-CF28BA653748}"
	MSIPackedCode  = "73CBF1BE79B05FC43A92FC82AB567384"
)
