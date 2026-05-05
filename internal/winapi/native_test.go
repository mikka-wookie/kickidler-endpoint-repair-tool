//go:build windows

package winapi

import (
	"context"
	"testing"
)

func TestRegistryUnsupportedRootIsStructured(t *testing.T) {
	got := QueryRegistryKey(context.Background(), "HKXX", `Software\Test`, RegistryViewDefault)
	if !got.UnsupportedRoot {
		t.Fatalf("UnsupportedRoot = false, want true: %#v", got)
	}
	if got.Error == "" {
		t.Fatalf("expected support-readable error")
	}
}

func TestNonexistentRegistryKeyIsNotFatal(t *testing.T) {
	got := QueryRegistryKey(context.Background(), "HKCU", `Software\kigrepair-test-key-that-should-not-exist`, RegistryViewDefault)
	if got.Exists {
		t.Fatalf("Exists = true, want false")
	}
	if got.Error != "" {
		t.Fatalf("Error = %q, want empty for missing key", got.Error)
	}
}
