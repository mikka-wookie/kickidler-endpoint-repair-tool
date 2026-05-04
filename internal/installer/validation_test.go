package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/config"
)

func TestValidateMSIBasicChecks(t *testing.T) {
	dir := t.TempDir()
	msi := writeValidationFile(t, dir, "grabberEM.x64.msi", []byte("msi bytes"))
	txt := writeValidationFile(t, dir, "wrong.txt", []byte("not msi"))
	unknown := writeValidationFile(t, dir, "custom installer.msi", []byte("msi bytes"))
	directory := filepath.Join(dir, "folder.msi")
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		path          string
		opts          ValidationOptions
		wantStatus    string
		wantSupported bool
		wantArch      string
		wantWarning   string
	}{
		{name: "empty installer path", path: "", opts: testValidationOptions(true), wantStatus: ValidationStatusInvalid},
		{name: "missing file", path: filepath.Join(dir, "missing.msi"), opts: testValidationOptions(true), wantStatus: ValidationStatusInvalid},
		{name: "directory path", path: directory, opts: testValidationOptions(true), wantStatus: ValidationStatusInvalid},
		{name: "non MSI extension", path: txt, opts: testValidationOptions(true), wantStatus: ValidationStatusInvalid},
		{name: "supported filename", path: msi, opts: testValidationOptions(true), wantStatus: ValidationStatusValid, wantSupported: true, wantArch: "x64"},
		{name: "unsupported explicit filename warns", path: unknown, opts: testValidationOptions(true), wantStatus: ValidationStatusValidWithWarnings, wantWarning: "supported_filename"},
		{name: "unsupported auto filename fails", path: unknown, opts: testValidationOptions(false), wantStatus: ValidationStatusInvalid},
		{name: "path with spaces", path: unknown, opts: testValidationOptions(true), wantStatus: ValidationStatusValidWithWarnings},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateMSI(tt.path, tt.opts)
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s; checks=%#v errors=%#v warnings=%#v", got.Status, tt.wantStatus, got.Checks, got.Errors, got.Warnings)
			}
			if tt.wantSupported && !got.SupportedName {
				t.Fatal("supported name was not detected")
			}
			if tt.wantArch != "" && got.Architecture != tt.wantArch {
				t.Fatalf("architecture = %q, want %q", got.Architecture, tt.wantArch)
			}
			if tt.wantWarning != "" && !hasValidationWarning(got, tt.wantWarning) {
				t.Fatalf("warnings = %#v, want code %q", got.Warnings, tt.wantWarning)
			}
		})
	}
}

func TestValidateMSIArchitectureFromName(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name       string
		wantArch   string
		wantStatus string
	}{
		{name: "grabberEM.x64.msi", wantArch: "x64", wantStatus: ValidationStatusValid},
		{name: "grabberTT.x32.msi", wantArch: "x32", wantStatus: ValidationStatusValid},
		{name: "grabber.msi", wantArch: "unknown", wantStatus: ValidationStatusValidWithWarnings},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeValidationFile(t, dir, tt.name, []byte(tt.name))
			got := ValidateMSI(path, testValidationOptions(true))
			if got.Architecture != tt.wantArch {
				t.Fatalf("architecture = %q, want %q", got.Architecture, tt.wantArch)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", got.Status, tt.wantStatus)
			}
		})
	}
}

func TestValidateMSIComputesSHA256(t *testing.T) {
	content := []byte("known content")
	path := writeValidationFile(t, t.TempDir(), "grabberEM.x64.msi", content)
	got := ValidateMSI(path, testValidationOptions(true))
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	if got.SHA256 != want {
		t.Fatalf("sha256 = %q, want %q", got.SHA256, want)
	}
}

func TestValidateMSIMetadataOutcomes(t *testing.T) {
	path := writeValidationFile(t, t.TempDir(), "grabberEM.x64.msi", []byte("msi"))

	t.Run("product code mismatch fails when metadata read succeeds", func(t *testing.T) {
		opts := testValidationOptions(true)
		opts.DeepMetadata = true
		got := validateMSI(path, opts, validationRuntime{
			readMSIMetadata: func(string) (msiMetadata, error) {
				return msiMetadata{ProductCode: "{11111111-1111-1111-1111-111111111111}"}, nil
			},
		})
		if got.Status != ValidationStatusInvalid {
			t.Fatalf("status = %s, want invalid", got.Status)
		}
		if !hasValidationCheck(got, "msi_product_code", ValidationCheckFail) {
			t.Fatalf("checks = %#v, want product code failure", got.Checks)
		}
	})

	t.Run("metadata unavailable warns only", func(t *testing.T) {
		opts := testValidationOptions(true)
		opts.DeepMetadata = true
		got := validateMSI(path, opts, validationRuntime{
			readMSIMetadata: func(string) (msiMetadata, error) {
				return msiMetadata{}, errors.New("COM unavailable")
			},
		})
		if got.Status != ValidationStatusValidWithWarnings {
			t.Fatalf("status = %s, want valid_with_warnings", got.Status)
		}
		if !hasValidationCheck(got, "msi_product_code", ValidationCheckWarning) {
			t.Fatalf("checks = %#v, want product code warning", got.Checks)
		}
	})

	t.Run("signature unavailable warns only", func(t *testing.T) {
		opts := testValidationOptions(true)
		opts.SignatureCheck = true
		got := validateMSI(path, opts, validationRuntime{
			checkSignature: func(string) (SignatureResult, error) {
				return SignatureResult{Checked: true, Status: "unavailable", Error: "blocked"}, errors.New("blocked")
			},
		})
		if got.Status != ValidationStatusValidWithWarnings {
			t.Fatalf("status = %s, want valid_with_warnings", got.Status)
		}
		if !hasValidationCheck(got, "signature", ValidationCheckWarning) {
			t.Fatalf("checks = %#v, want signature warning", got.Checks)
		}
	})
}

func TestValidationDoesNotContainInvite(t *testing.T) {
	secret := "SECRET-INVITE-999"
	path := writeValidationFile(t, t.TempDir(), "grabberEM.x64.msi", []byte(secret))
	got := ValidateMSI(path, testValidationOptions(true))
	serialized := strings.Join(append(append([]string{}, got.Warnings...), got.Errors...), "\n")
	for _, check := range got.Checks {
		serialized += "\n" + check.Evidence + "\n" + check.Action
	}
	if strings.Contains(serialized, secret) {
		t.Fatal("validation result contains raw invite")
	}
}

func testValidationOptions(explicit bool) ValidationOptions {
	return ValidationOptions{
		ExplicitPath:        explicit,
		ExpectedProductCode: config.GrabberMSIProductCode,
		ExpectedPackedCode:  config.GrabberMSIPackedCode,
		PreferredArch:       "",
		AllowUnknownName:    explicit,
		DeepMetadata:        false,
		SignatureCheck:      false,
	}
}

func writeValidationFile(t *testing.T, dir string, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func hasValidationWarning(result ValidationResult, code string) bool {
	return hasValidationCheck(result, code, ValidationCheckWarning)
}

func hasValidationCheck(result ValidationResult, code string, status string) bool {
	for _, check := range result.Checks {
		if check.Code == code && check.Status == status {
			return true
		}
	}
	return false
}
