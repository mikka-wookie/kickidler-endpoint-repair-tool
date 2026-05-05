package detector

import (
	"context"

	"kigrepair/internal/winapi"
)

type RegistryState struct {
	Root   string `json:"root"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

type registryTarget struct {
	rootName string
	path     string
	view     winapi.RegistryView
}

func DetectRegistry() []RegistryState {
	targets := []registryTarget{
		{"HKCU", `Software\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`, winapi.RegistryViewDefault},
		{"HKLM", `SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`, winapi.RegistryViewDefault},
		{"HKLM", `SOFTWARE\WOW6432Node\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`, winapi.RegistryView32},
		{"HKCR", `Installer\Features\73CBF1BE79B05FC43A92FC82AB567384`, winapi.RegistryViewDefault},
		{"HKCR", `Installer\Products\73CBF1BE79B05FC43A92FC82AB567384`, winapi.RegistryViewDefault},
		{"HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`, winapi.RegistryViewDefault},
		{"HKLM", `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`, winapi.RegistryView32},
		{"HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products\73CBF1BE79B05FC43A92FC82AB567384`, winapi.RegistryViewDefault},
	}
	states := make([]RegistryState, 0, len(targets))
	for _, target := range targets {
		states = append(states, detectRegistryKey(target))
	}
	return states
}

func detectRegistryKey(target registryTarget) RegistryState {
	native := winapi.QueryRegistryKey(context.Background(), target.rootName, target.path, target.view)
	return RegistryState{Root: native.Root, Path: native.Path, Exists: native.Exists, Error: native.Error}
}
