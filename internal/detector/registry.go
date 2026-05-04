package detector

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

type RegistryState struct {
	Root   string `json:"root"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

type registryTarget struct {
	rootName string
	root     registry.Key
	path     string
}

func DetectRegistry() []RegistryState {
	targets := []registryTarget{
		{"HKCU", registry.CURRENT_USER, `Software\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`},
		{"HKLM", registry.LOCAL_MACHINE, `SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`},
		{"HKLM", registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`},
		{"HKCR", registry.CLASSES_ROOT, `Installer\Features\73CBF1BE79B05FC43A92FC82AB567384`},
		{"HKCR", registry.CLASSES_ROOT, `Installer\Products\73CBF1BE79B05FC43A92FC82AB567384`},
		{"HKLM", registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`},
		{"HKLM", registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`},
		{"HKLM", registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products\73CBF1BE79B05FC43A92FC82AB567384`},
	}
	states := make([]RegistryState, 0, len(targets))
	for _, target := range targets {
		states = append(states, detectRegistryKey(target))
	}
	return states
}

func detectRegistryKey(target registryTarget) RegistryState {
	state := RegistryState{Root: target.rootName, Path: target.path}
	key, err := registry.OpenKey(target.root, target.path, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			state.Exists = false
			return state
		}
		state.Error = err.Error()
		return state
	}
	key.Close()
	state.Exists = true
	return state
}
