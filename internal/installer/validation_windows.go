//go:build windows

package installer

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func readMSIMetadata(path string) (msiMetadata, error) {
	script := `
$ErrorActionPreference = 'Stop'
$Path = $env:KIGREPAIR_MSI_PATH
$installer = New-Object -ComObject WindowsInstaller.Installer
$db = $installer.OpenDatabase($Path, 0)
$view = $db.OpenView("SELECT [Value] FROM [Property] WHERE [Property] = 'ProductCode'")
$view.Execute()
$record = $view.Fetch()
$productCode = ""
if ($null -ne $record) { $productCode = [string]$record.StringData(1) }
[pscustomobject]@{ ProductCode = $productCode } | ConvertTo-Json -Compress
`
	output, err := runValidationPowerShell(path, script)
	if err != nil {
		return msiMetadata{}, err
	}
	var parsed struct {
		ProductCode string `json:"ProductCode"`
	}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return msiMetadata{}, err
	}
	return msiMetadata{ProductCode: strings.TrimSpace(parsed.ProductCode)}, nil
}

func checkMSISignature(path string) (SignatureResult, error) {
	script := `
$ErrorActionPreference = 'Stop'
$Path = $env:KIGREPAIR_MSI_PATH
$sig = Get-AuthenticodeSignature -FilePath $Path
$subject = ""
$issuer = ""
$thumbprint = ""
if ($null -ne $sig.SignerCertificate) {
  $subject = [string]$sig.SignerCertificate.Subject
  $issuer = [string]$sig.SignerCertificate.Issuer
  $thumbprint = [string]$sig.SignerCertificate.Thumbprint
}
[pscustomobject]@{
  Status = [string]$sig.Status
  StatusMessage = [string]$sig.StatusMessage
  Subject = $subject
  Issuer = $issuer
  Thumbprint = $thumbprint
} | ConvertTo-Json -Compress
`
	output, err := runValidationPowerShell(path, script)
	if err != nil {
		return SignatureResult{Checked: true, Status: "unavailable", Error: err.Error()}, err
	}
	var parsed struct {
		Status        string `json:"Status"`
		StatusMessage string `json:"StatusMessage"`
		Subject       string `json:"Subject"`
		Issuer        string `json:"Issuer"`
		Thumbprint    string `json:"Thumbprint"`
	}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return SignatureResult{Checked: true, Status: "unavailable", Error: err.Error()}, err
	}
	status := strings.ToLower(strings.TrimSpace(parsed.Status))
	switch status {
	case "":
		status = "unknown"
	case "valid":
		status = "valid"
	case "nottrusted", "hashmismatch", "notvalidforsignature", "notvalidfortimestamp", "incompatible":
		status = "invalid"
	default:
		status = "unknown"
	}
	return SignatureResult{
		Checked:    true,
		Status:     status,
		Subject:    parsed.Subject,
		Issuer:     parsed.Issuer,
		Thumbprint: parsed.Thumbprint,
		Error:      parsed.StatusMessage,
	}, nil
}

func runValidationPowerShell(path string, script string) (string, error) {
	command := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	command.Env = append(os.Environ(), "KIGREPAIR_MSI_PATH="+path)
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text != "" {
			return "", errors.New(text)
		}
		return "", err
	}
	return text, nil
}
