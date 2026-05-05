//go:build !windows

package winapi

import (
	"context"
	"fmt"
)

type defaultNativeServiceProvider struct{}

func (defaultNativeServiceProvider) QueryService(ctx context.Context, name string) (NativeServiceInfo, error) {
	return NativeServiceInfo{Name: name, Error: ErrUnsupportedPlatform.Error()}, ErrUnsupportedPlatform
}

func (defaultNativeServiceProvider) StopService(ctx context.Context, name string) error {
	return fmt.Errorf("%w: service stop is Windows-only", ErrUnsupportedPlatform)
}

func (defaultNativeServiceProvider) DeleteService(ctx context.Context, name string) error {
	return fmt.Errorf("%w: service delete is Windows-only", ErrUnsupportedPlatform)
}
