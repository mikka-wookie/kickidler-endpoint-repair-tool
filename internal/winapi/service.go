package winapi

import (
	"context"
	"errors"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

type NativeServiceInfo struct {
	Name            string
	Exists          bool
	State           string
	StartType       string
	BinaryPathName  string
	ServiceType     uint32
	PermissionError bool
	Error           string
}

type NativeServiceProvider interface {
	QueryService(ctx context.Context, name string) (NativeServiceInfo, error)
	StopService(ctx context.Context, name string) error
	DeleteService(ctx context.Context, name string) error
}

func QueryServiceNative(ctx context.Context, name string) (NativeServiceInfo, error) {
	return defaultNativeServiceProvider{}.QueryService(ctx, name)
}

func StopServiceNative(ctx context.Context, name string) error {
	return defaultNativeServiceProvider{}.StopService(ctx, name)
}

func DeleteServiceNative(ctx context.Context, name string) error {
	return defaultNativeServiceProvider{}.DeleteService(ctx, name)
}

func WindowsVersion() string {
	return windowsVersion()
}
