//go:build windows

package winapi

import (
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

func TestMapServiceState(t *testing.T) {
	tests := []struct {
		state svc.State
		want  string
	}{
		{svc.Running, "running"},
		{svc.Stopped, "stopped"},
		{svc.StartPending, "start_pending"},
		{svc.StopPending, "stop_pending"},
		{svc.Paused, "paused"},
		{svc.PausePending, "pause_pending"},
		{svc.ContinuePending, "continue_pending"},
		{svc.State(255), "unknown"},
	}
	for _, tt := range tests {
		if got := MapServiceState(tt.state); got != tt.want {
			t.Fatalf("MapServiceState(%v) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestMapServiceStartType(t *testing.T) {
	tests := []struct {
		startType uint32
		want      string
	}{
		{windows.SERVICE_AUTO_START, "automatic"},
		{windows.SERVICE_DEMAND_START, "manual"},
		{windows.SERVICE_DISABLED, "disabled"},
		{windows.SERVICE_BOOT_START, "boot"},
		{windows.SERVICE_SYSTEM_START, "system"},
		{255, "unknown"},
	}
	for _, tt := range tests {
		if got := MapServiceStartType(tt.startType); got != tt.want {
			t.Fatalf("MapServiceStartType(%d) = %q, want %q", tt.startType, got, tt.want)
		}
	}
}
