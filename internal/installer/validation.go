package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kigrepair/internal/config"
	"kigrepair/internal/detector"
)

const (
	ValidationStatusValid             = "valid"
	ValidationStatusValidWithWarnings = "valid_with_warnings"
	ValidationStatusInvalid           = "invalid"
	ValidationStatusUnknown           = "unknown"

	ValidationCheckPass    = "pass"
	ValidationCheckWarning = "warning"
	ValidationCheckFail    = "fail"
	ValidationCheckSkipped = "skipped"
)

type ValidationOptions struct {
	ExplicitPath        bool
	ExpectedProductCode string
	ExpectedPackedCode  string
	PreferredArch       string
	AllowUnknownName    bool
	DeepMetadata        bool
	SignatureCheck      bool
}

type ValidationResult struct {
	Status                string            `json:"status"`
	Path                  string            `json:"path"`
	FileName              string            `json:"file_name"`
	Exists                bool              `json:"exists"`
	IsFile                bool              `json:"is_file"`
	Readable              bool              `json:"readable"`
	ExtensionOK           bool              `json:"extension_ok"`
	SupportedName         bool              `json:"supported_name"`
	Architecture          string            `json:"architecture,omitempty"`
	PreferredArchitecture bool              `json:"preferred_architecture"`
	SizeBytes             int64             `json:"size_bytes,omitempty"`
	SHA256                string            `json:"sha256,omitempty"`
	ProductCode           string            `json:"product_code,omitempty"`
	ProductCodeMatches    bool              `json:"product_code_matches"`
	PackedCodeMatches     bool              `json:"packed_code_matches"`
	Signature             *SignatureResult  `json:"signature,omitempty"`
	Checks                []ValidationCheck `json:"checks"`
	Warnings              []string          `json:"warnings,omitempty"`
	Errors                []string          `json:"errors,omitempty"`
}

type ValidationCheck struct {
	Code     string `json:"code"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Title    string `json:"title"`
	Evidence string `json:"evidence,omitempty"`
	Action   string `json:"action,omitempty"`
}

type SignatureResult struct {
	Checked    bool   `json:"checked"`
	Status     string `json:"status,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Issuer     string `json:"issuer,omitempty"`
	Thumbprint string `json:"thumbprint,omitempty"`
	Error      string `json:"error,omitempty"`
}

type msiMetadata struct {
	ProductCode string
}

type validationRuntime struct {
	readMSIMetadata func(string) (msiMetadata, error)
	checkSignature  func(string) (SignatureResult, error)
}

func ValidateMSI(path string, opts ValidationOptions) ValidationResult {
	return validateMSI(path, opts, validationRuntime{
		readMSIMetadata: readMSIMetadata,
		checkSignature:  checkMSISignature,
	})
}

func defaultValidationOptions(explicit bool, preferredArch string) ValidationOptions {
	return ValidationOptions{
		ExplicitPath:        explicit,
		ExpectedProductCode: config.GrabberMSIProductCode,
		ExpectedPackedCode:  config.GrabberMSIPackedCode,
		PreferredArch:       preferredArch,
		AllowUnknownName:    explicit,
		DeepMetadata:        true,
		SignatureCheck:      true,
	}
}

func validateMSI(path string, opts ValidationOptions, runtime validationRuntime) ValidationResult {
	opts.ExpectedProductCode = firstNonEmptyString(opts.ExpectedProductCode, config.GrabberMSIProductCode)
	opts.ExpectedPackedCode = firstNonEmptyString(opts.ExpectedPackedCode, config.GrabberMSIPackedCode)
	opts.PreferredArch = normalizeInstallerArch(opts.PreferredArch)
	if opts.ExplicitPath {
		opts.AllowUnknownName = true
	}

	trimmed := strings.TrimSpace(path)
	result := ValidationResult{Status: ValidationStatusUnknown, Path: trimmed}
	if trimmed == "" {
		result.addCheck("path_provided", ValidationCheckFail, true, "Installer path was not provided", "", `Provide --installer ".\grabberEM.x64.msi".`)
		result.finalize()
		return result
	}
	result.addCheck("path_provided", ValidationCheckPass, true, "Installer path was provided", trimmed, "")

	abs, err := filepath.Abs(trimmed)
	if err == nil {
		trimmed = detector.NormalizeWindowsPath(abs)
		result.Path = trimmed
	}
	result.FileName = filepath.Base(trimmed)

	info, err := os.Stat(trimmed)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.addCheck("file_exists", ValidationCheckFail, true, "Installer file exists", "file missing", "Use a supported Grabber MSI installer.")
		} else {
			result.addCheck("file_exists", ValidationCheckFail, true, "Installer file exists", err.Error(), "Verify the path and permissions.")
		}
		result.finalize()
		return result
	}
	result.Exists = true
	result.SizeBytes = info.Size()
	result.addCheck("file_exists", ValidationCheckPass, true, "Installer file exists", trimmed, "")

	result.IsFile = !info.IsDir() && info.Mode().IsRegular()
	if !result.IsFile {
		result.addCheck("is_file", ValidationCheckFail, true, "Installer path is a regular file", "path is not a regular file", "Select a supported Grabber MSI file.")
		result.finalize()
		return result
	}
	result.addCheck("is_file", ValidationCheckPass, true, "Installer path is a regular file", "", "")

	if file, err := os.Open(trimmed); err != nil {
		result.addCheck("readable", ValidationCheckFail, true, "Installer file is readable", err.Error(), "Verify file permissions.")
		result.finalize()
		return result
	} else {
		result.Readable = true
		_ = file.Close()
		result.addCheck("readable", ValidationCheckPass, true, "Installer file is readable", "", "")
	}

	result.ExtensionOK = strings.EqualFold(filepath.Ext(trimmed), ".msi")
	if !result.ExtensionOK {
		result.addCheck("extension_msi", ValidationCheckFail, true, "File is an MSI package", filepath.Ext(trimmed), "Use a .msi Grabber installer.")
		result.finalize()
		return result
	}
	result.addCheck("extension_msi", ValidationCheckPass, true, "File is an MSI package", ".msi", "")

	_, arch, supported := ParseInstallerName(result.FileName)
	result.SupportedName = supported
	if supported {
		result.addCheck("supported_filename", ValidationCheckPass, !opts.ExplicitPath, "Installer filename is supported", result.FileName, "")
	} else if opts.AllowUnknownName {
		result.addCheck("supported_filename", ValidationCheckWarning, !opts.ExplicitPath, "Installer filename is not in the supported name list", result.FileName, "Confirm this is the correct Grabber MSI.")
	} else {
		result.addCheck("supported_filename", ValidationCheckFail, true, "Installer filename is supported", result.FileName, "Use a supported Grabber MSI filename.")
		result.finalize()
		return result
	}

	if arch == "" {
		arch = architectureFromName(result.FileName)
	}
	result.Architecture = arch
	if opts.PreferredArch == "" || arch == opts.PreferredArch {
		result.PreferredArchitecture = opts.PreferredArch != ""
	}
	switch {
	case arch == "unknown":
		result.addCheck("architecture_from_name", ValidationCheckWarning, false, "Installer architecture was detected from filename", result.FileName, "Prefer an x64 or x32 Grabber MSI when available.")
	case opts.PreferredArch != "" && arch != opts.PreferredArch:
		result.addCheck("architecture_from_name", ValidationCheckWarning, false, "Installer architecture differs from preferred architecture", fmt.Sprintf("detected=%s preferred=%s", arch, opts.PreferredArch), "Use the preferred architecture MSI if available.")
	default:
		result.addCheck("architecture_from_name", ValidationCheckPass, false, "Installer architecture was detected from filename", arch, "")
	}

	if sum, err := sha256FileStreaming(trimmed); err != nil {
		result.addCheck("sha256", ValidationCheckWarning, false, "SHA-256 hash was calculated", err.Error(), "Hash could not be calculated.")
	} else {
		result.SHA256 = sum
		result.addCheck("sha256", ValidationCheckPass, false, "SHA-256 hash was calculated", sum, "")
	}

	if opts.DeepMetadata {
		result.validateMetadata(trimmed, opts, runtime.readMSIMetadata)
	} else {
		result.addCheck("msi_product_code", ValidationCheckSkipped, false, "MSI ProductCode metadata check", "deep metadata disabled", "")
		result.addCheck("msi_packed_code", ValidationCheckSkipped, false, "MSI packed product code check", "deep metadata disabled", "")
	}

	if opts.SignatureCheck {
		result.validateSignature(trimmed, runtime.checkSignature)
	} else {
		result.addCheck("signature", ValidationCheckSkipped, false, "Authenticode signature check", "signature check disabled", "")
	}

	result.finalize()
	return result
}

func (r *ValidationResult) validateMetadata(path string, opts ValidationOptions, read func(string) (msiMetadata, error)) {
	if read == nil {
		r.addCheck("msi_product_code", ValidationCheckWarning, false, "MSI ProductCode metadata check", "metadata reader unavailable", "")
		r.addCheck("msi_packed_code", ValidationCheckSkipped, false, "MSI packed product code check", "metadata reader unavailable", "")
		return
	}
	metadata, err := read(path)
	if err != nil {
		r.addCheck("msi_product_code", ValidationCheckWarning, false, "MSI ProductCode metadata check", err.Error(), "Metadata could not be read on this system.")
		r.addCheck("msi_packed_code", ValidationCheckSkipped, false, "MSI packed product code check", "ProductCode unavailable", "")
		return
	}
	r.ProductCode = strings.TrimSpace(metadata.ProductCode)
	if r.ProductCode == "" {
		r.addCheck("msi_product_code", ValidationCheckWarning, false, "MSI ProductCode metadata check", "ProductCode missing", "Confirm the installer source.")
		r.addCheck("msi_packed_code", ValidationCheckSkipped, false, "MSI packed product code check", "ProductCode missing", "")
		return
	}
	r.ProductCodeMatches = strings.EqualFold(r.ProductCode, opts.ExpectedProductCode)
	if !r.ProductCodeMatches {
		r.addCheck("msi_product_code", ValidationCheckFail, true, "MSI ProductCode matches expected Grabber product code", r.ProductCode, "Use the supported Grabber MSI installer.")
		r.addCheck("msi_packed_code", ValidationCheckSkipped, false, "MSI packed product code check", "ProductCode mismatch", "")
		return
	}
	r.addCheck("msi_product_code", ValidationCheckPass, true, "MSI ProductCode matches expected Grabber product code", r.ProductCode, "")

	packed := packProductCode(r.ProductCode)
	r.PackedCodeMatches = strings.EqualFold(packed, opts.ExpectedPackedCode)
	if r.PackedCodeMatches {
		r.addCheck("msi_packed_code", ValidationCheckPass, false, "MSI packed product code matches expected Grabber packed code", packed, "")
	} else {
		r.addCheck("msi_packed_code", ValidationCheckWarning, false, "MSI packed product code could not be confirmed", packed, "Confirm installer metadata if package source is uncertain.")
	}
}

func (r *ValidationResult) validateSignature(path string, check func(string) (SignatureResult, error)) {
	if check == nil {
		r.addCheck("signature", ValidationCheckWarning, false, "Authenticode signature check", "signature checker unavailable", "")
		return
	}
	signature, err := check(path)
	signature.Checked = true
	r.Signature = &signature
	if err != nil {
		r.Signature.Status = firstNonEmptyString(r.Signature.Status, "unavailable")
		r.Signature.Error = firstNonEmptyString(r.Signature.Error, err.Error())
		r.addCheck("signature", ValidationCheckWarning, false, "Authenticode signature check", r.Signature.Error, "Signature check could not be completed on this system.")
		return
	}
	switch strings.ToLower(strings.TrimSpace(signature.Status)) {
	case "valid":
		r.addCheck("signature", ValidationCheckPass, false, "Authenticode signature is valid", signature.Subject, "")
	case "invalid":
		r.addCheck("signature", ValidationCheckWarning, false, "Authenticode signature is invalid", signature.Error, "Confirm installer source before installing.")
	default:
		evidence := firstNonEmptyString(signature.Status, signature.Error, "unknown")
		r.addCheck("signature", ValidationCheckWarning, false, "Authenticode signature status is not valid", evidence, "Confirm installer source before installing.")
	}
}

func (r *ValidationResult) addCheck(code string, status string, required bool, title string, evidence string, action string) {
	check := ValidationCheck{Code: code, Status: status, Required: required, Title: title, Evidence: evidence, Action: action}
	r.Checks = append(r.Checks, check)
	switch status {
	case ValidationCheckFail:
		if evidence != "" {
			r.Errors = append(r.Errors, code+": "+evidence)
		} else {
			r.Errors = append(r.Errors, code+": "+title)
		}
	case ValidationCheckWarning:
		if evidence != "" {
			r.Warnings = append(r.Warnings, code+": "+evidence)
		} else {
			r.Warnings = append(r.Warnings, code+": "+title)
		}
	}
}

func (r *ValidationResult) finalize() {
	hasRequiredFailure := false
	hasWarning := false
	for _, check := range r.Checks {
		if check.Status == ValidationCheckFail && check.Required {
			hasRequiredFailure = true
		}
		if check.Status == ValidationCheckWarning {
			hasWarning = true
		}
	}
	switch {
	case hasRequiredFailure:
		r.Status = ValidationStatusInvalid
	case hasWarning:
		r.Status = ValidationStatusValidWithWarnings
	case len(r.Checks) == 0:
		r.Status = ValidationStatusUnknown
	default:
		r.Status = ValidationStatusValid
	}
}

func (r ValidationResult) IsUsable() bool {
	return r.Status == ValidationStatusValid || r.Status == ValidationStatusValidWithWarnings
}

func (r ValidationResult) ErrorSummary() string {
	for _, check := range r.Checks {
		if check.Status == ValidationCheckFail {
			if check.Action != "" {
				return check.Title + ": " + check.Evidence
			}
			return firstNonEmptyString(check.Evidence, check.Title)
		}
	}
	if len(r.Errors) > 0 {
		return r.Errors[0]
	}
	return "installer validation failed"
}

func architectureFromName(name string) string {
	lower := strings.ToLower(filepath.Base(name))
	switch {
	case strings.Contains(lower, ".x64."):
		return "x64"
	case strings.Contains(lower, ".x32."):
		return "x32"
	case strings.EqualFold(lower, "grabber.msi"):
		return "unknown"
	default:
		return "unknown"
	}
}

func normalizeInstallerArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "amd64", "x64":
		return "x64"
	case "386", "x86", "x32":
		return "x32"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

func osPreferredInstallerArch(osArch string) string {
	switch normalizeOSArchitecture(osArch) {
	case "386":
		return "x32"
	default:
		return "x64"
	}
}

func sha256FileStreaming(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func packProductCode(productCode string) string {
	cleaned := strings.ToUpper(strings.Trim(productCode, "{}"))
	parts := strings.Split(cleaned, "-")
	if len(parts) != 5 {
		return ""
	}
	return reverseRunes(parts[0]) +
		reverseRunes(parts[1]) +
		reverseRunes(parts[2]) +
		swapPairs(parts[3]) +
		swapPairs(parts[4])
}

func reverseRunes(value string) string {
	runes := []rune(value)
	for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
		runes[left], runes[right] = runes[right], runes[left]
	}
	return string(runes)
}

func swapPairs(value string) string {
	if len(value)%2 != 0 {
		return value
	}
	var b strings.Builder
	for i := 0; i < len(value); i += 2 {
		b.WriteByte(value[i+1])
		b.WriteByte(value[i])
	}
	return b.String()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
