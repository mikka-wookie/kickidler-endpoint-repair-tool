package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInvite(t *testing.T) {
	tests := []struct {
		name    string
		invite  string
		wantErr bool
	}{
		{name: "empty", invite: "", wantErr: true},
		{name: "whitespace", invite: "   ", wantErr: true},
		{name: "normal", invite: "ABC123", wantErr: false},
		{name: "quote", invite: `ABC"123`, wantErr: true},
		{name: "ampersand", invite: "ABC&123", wantErr: true},
		{name: "pipe", invite: "ABC|123", wantErr: true},
		{name: "semicolon", invite: "ABC;123", wantErr: true},
		{name: "greater", invite: "ABC>123", wantErr: true},
		{name: "less", invite: "ABC<123", wantErr: true},
		{name: "backtick", invite: "ABC`123", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateInvite(tt.invite)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateInvite() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstallerPath(t *testing.T) {
	dir := t.TempDir()
	msiPath := filepath.Join(dir, "grabber.msi")
	if err := os.WriteFile(msiPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	txtPath := filepath.Join(dir, "grabber.txt")
	if err := os.WriteFile(txtPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "missing path", path: "", wantErr: true},
		{name: "non-msi", path: txtPath, wantErr: true},
		{name: "nonexistent", path: filepath.Join(dir, "missing.msi"), wantErr: true},
		{name: "existing msi", path: msiPath, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateInstallerPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateInstallerPath() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}
