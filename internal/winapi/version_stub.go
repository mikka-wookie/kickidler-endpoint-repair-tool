//go:build !windows

package winapi

func windowsVersion() string {
	return ""
}
