//go:build windows

package winapi

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type defaultNativeServiceProvider struct{}

func (defaultNativeServiceProvider) QueryService(ctx context.Context, name string) (NativeServiceInfo, error) {
	if err := ctx.Err(); err != nil {
		return NativeServiceInfo{Name: name, Error: err.Error()}, err
	}
	m, err := mgr.Connect()
	if err != nil {
		return NativeServiceInfo{Name: name, Error: err.Error(), PermissionError: errors.Is(err, windows.ERROR_ACCESS_DENIED)}, err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		info := NativeServiceInfo{Name: name}
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			info.Exists = false
			return info, nil
		}
		info.Error = err.Error()
		info.PermissionError = errors.Is(err, windows.ERROR_ACCESS_DENIED)
		return info, err
	}
	defer s.Close()

	status, statusErr := s.Query()
	cfg, cfgErr := s.Config()
	info := NativeServiceInfo{Name: name, Exists: true}
	if statusErr == nil {
		info.State = MapServiceState(status.State)
	}
	if cfgErr == nil {
		info.StartType = MapServiceStartType(cfg.StartType)
		info.BinaryPathName = cfg.BinaryPathName
		info.ServiceType = cfg.ServiceType
	}
	if statusErr != nil || cfgErr != nil {
		errText := firstError(statusErr, cfgErr).Error()
		info.Error = errText
		info.PermissionError = errors.Is(statusErr, windows.ERROR_ACCESS_DENIED) || errors.Is(cfgErr, windows.ERROR_ACCESS_DENIED)
		return info, fmt.Errorf("native service query failed for %s: %w", name, firstError(statusErr, cfgErr))
	}
	return info, nil
}

func (defaultNativeServiceProvider) StopService(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return nil
		}
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	if errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return nil
	}
	return err
}

func (defaultNativeServiceProvider) DeleteService(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return nil
		}
		return err
	}
	defer s.Close()
	return s.Delete()
}

func MapServiceState(state svc.State) string {
	switch state {
	case svc.Running:
		return "running"
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "start_pending"
	case svc.StopPending:
		return "stop_pending"
	case svc.Paused:
		return "paused"
	case svc.PausePending:
		return "pause_pending"
	case svc.ContinuePending:
		return "continue_pending"
	default:
		return "unknown"
	}
}

func MapServiceStartType(startType uint32) string {
	switch startType {
	case windows.SERVICE_AUTO_START:
		return "automatic"
	case windows.SERVICE_DEMAND_START:
		return "manual"
	case windows.SERVICE_DISABLED:
		return "disabled"
	case windows.SERVICE_BOOT_START:
		return "boot"
	case windows.SERVICE_SYSTEM_START:
		return "system"
	default:
		return "unknown"
	}
}

func firstError(left, right error) error {
	if left != nil {
		return left
	}
	return right
}
