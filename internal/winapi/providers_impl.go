package winapi

import "context"

type defaultRegistryProvider struct{}

func (defaultRegistryProvider) QueryKey(root string, path string, view RegistryView) RegistryKeyResult {
	return QueryRegistryKey(context.Background(), root, path, view)
}

func (defaultRegistryProvider) DeleteTree(root string, path string) error {
	return DeleteRegistryTree(context.Background(), root, path)
}

type defaultProcessProvider struct{}

func (defaultProcessProvider) ListProcesses() ([]RawProcessInfo, error) {
	return ListProcesses(context.Background())
}

func (defaultProcessProvider) QueryProcess(pid int) (*RawProcessInfo, error) {
	return QueryProcess(context.Background(), pid)
}

func (defaultProcessProvider) TerminateProcess(pid int) error {
	return TerminateProcessByPID(context.Background(), pid)
}

type defaultTokenProvider struct{}

func (defaultTokenProvider) IsAdmin() (bool, error) {
	return IsAdmin(), nil
}
