package winapi

import "context"

type RawProcessInfo struct {
	ProcessID      int
	Name           string
	ExecutablePath string
	CommandLine    string
}

func ListProcesses(ctx context.Context) ([]RawProcessInfo, error) {
	return listProcesses(ctx)
}

func QueryProcess(ctx context.Context, pid int) (*RawProcessInfo, error) {
	processes, err := listProcesses(ctx)
	if err != nil {
		return nil, err
	}
	for _, process := range processes {
		if process.ProcessID == pid {
			return &process, nil
		}
	}
	return nil, nil
}

func TerminateProcessByPID(ctx context.Context, pid int) error {
	return terminateProcessByPID(ctx, pid)
}
