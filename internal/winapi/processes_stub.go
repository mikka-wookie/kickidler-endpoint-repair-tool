//go:build !windows

package winapi

import (
	"context"
	"fmt"
)

func listProcesses(ctx context.Context) ([]RawProcessInfo, error) {
	return nil, ErrUnsupportedPlatform
}

func terminateProcessByPID(ctx context.Context, pid int) error {
	return fmt.Errorf("%w: process termination is Windows-only", ErrUnsupportedPlatform)
}
