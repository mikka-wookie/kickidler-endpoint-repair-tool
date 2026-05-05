package winapi

type SystemProviders struct {
	Services  NativeServiceProvider
	Registry  RegistryProvider
	Processes ProcessProvider
	Tokens    TokenProvider
	Commands  CommandRunner
}

type RegistryProvider interface {
	QueryKey(root string, path string, view RegistryView) RegistryKeyResult
	DeleteTree(root string, path string) error
}

type ProcessProvider interface {
	ListProcesses() ([]RawProcessInfo, error)
	QueryProcess(pid int) (*RawProcessInfo, error)
	TerminateProcess(pid int) error
}

type TokenProvider interface {
	IsAdmin() (bool, error)
}

type CommandRunner func(CommandOptions) CommandResult

func NewSystemProviders() SystemProviders {
	return SystemProviders{
		Services:  defaultNativeServiceProvider{},
		Registry:  defaultRegistryProvider{},
		Processes: defaultProcessProvider{},
		Tokens:    defaultTokenProvider{},
		Commands:  RunCommand,
	}
}
