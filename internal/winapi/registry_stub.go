//go:build !windows

package winapi

import (
	"context"
	"fmt"
)

func queryRegistryKey(ctx context.Context, root string, path string, view RegistryView) RegistryKeyResult {
	return RegistryKeyResult{
		Root:             root,
		Path:             path,
		View:             string(view),
		Error:            ErrUnsupportedPlatform.Error(),
		SupportableError: "unsupported_os",
	}
}

func deleteRegistryTree(ctx context.Context, root string, path string) error {
	return fmt.Errorf("%w: registry deletion is Windows-only", ErrUnsupportedPlatform)
}
