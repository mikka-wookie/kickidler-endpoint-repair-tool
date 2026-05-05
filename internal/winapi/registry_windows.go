//go:build windows

package winapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func queryRegistryKey(ctx context.Context, rootName string, path string, view RegistryView) RegistryKeyResult {
	result := RegistryKeyResult{Root: rootName, Path: path, View: string(view)}
	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result
	}
	root, err := registryRoot(rootName)
	if err != nil {
		result.UnsupportedRoot = true
		result.Error = err.Error()
		return result
	}
	access := uint32(registry.QUERY_VALUE)
	switch view {
	case RegistryView32:
		access |= registry.WOW64_32KEY
	case RegistryView64:
		access |= registry.WOW64_64KEY
	}
	key, err := registry.OpenKey(root, path, access)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			result.Exists = false
			result.Accessible = true
			return result
		}
		result.Exists = true
		result.AccessDenied = errors.Is(err, windows.ERROR_ACCESS_DENIED)
		result.Error = err.Error()
		result.SupportableError = registrySupportableError(err)
		return result
	}
	key.Close()
	result.Exists = true
	result.Accessible = true
	return result
}

func deleteRegistryTree(ctx context.Context, rootName string, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := registryRoot(rootName)
	if err != nil {
		return err
	}
	return deleteRegistryTreeNative(root, path)
}

func deleteRegistryTreeNative(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE|registry.SET_VALUE)
	if err == nil {
		subkeys, readErr := key.ReadSubKeyNames(-1)
		key.Close()
		if readErr != nil {
			return readErr
		}
		for _, subkey := range subkeys {
			if err := deleteRegistryTreeNative(root, path+`\`+subkey); err != nil {
				return err
			}
		}
	}
	err = registry.DeleteKey(root, path)
	if errors.Is(err, registry.ErrNotExist) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return nil
	}
	return err
}

func registryRoot(root string) (registry.Key, error) {
	switch strings.ToUpper(strings.TrimSpace(root)) {
	case "HKCU", "HKEY_CURRENT_USER":
		return registry.CURRENT_USER, nil
	case "HKLM", "HKEY_LOCAL_MACHINE":
		return registry.LOCAL_MACHINE, nil
	case "HKCR", "HKEY_CLASSES_ROOT":
		return registry.CLASSES_ROOT, nil
	default:
		return 0, fmt.Errorf("unsupported registry root %q", root)
	}
}

func registrySupportableError(err error) string {
	switch {
	case errors.Is(err, windows.ERROR_ACCESS_DENIED):
		return "permission_denied"
	case errors.Is(err, windows.ERROR_INVALID_NAME):
		return "invalid_target"
	default:
		return "registry_access_failed"
	}
}
