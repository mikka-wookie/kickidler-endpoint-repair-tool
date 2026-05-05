//go:build windows

package winapi

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func windowsVersion() string {
	version := windows.RtlGetVersion()
	if version == nil {
		return os.Getenv("OS")
	}
	return fmt.Sprintf("%d.%d.%d", version.MajorVersion, version.MinorVersion, version.BuildNumber)
}
