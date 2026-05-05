package winapi

import "context"

type RegistryView string

const (
	RegistryViewDefault RegistryView = "default"
	RegistryView32      RegistryView = "32"
	RegistryView64      RegistryView = "64"
)

type RegistryKeyResult struct {
	Root             string `json:"root"`
	Path             string `json:"path"`
	View             string `json:"view,omitempty"`
	Exists           bool   `json:"exists"`
	Accessible       bool   `json:"accessible"`
	AccessDenied     bool   `json:"access_denied,omitempty"`
	UnsupportedRoot  bool   `json:"unsupported_root,omitempty"`
	Error            string `json:"error,omitempty"`
	SupportableError string `json:"supportable_error,omitempty"`
}

func QueryRegistryKey(ctx context.Context, root string, path string, view RegistryView) RegistryKeyResult {
	return queryRegistryKey(ctx, root, path, view)
}

func DeleteRegistryTree(ctx context.Context, root string, path string) error {
	return deleteRegistryTree(ctx, root, path)
}
