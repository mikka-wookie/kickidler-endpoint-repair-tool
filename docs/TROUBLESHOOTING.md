# kigrepair Troubleshooting

For configuration problems, start with:

```powershell
.\kigrepair.exe config validate --config ".\kigrepair.yaml"
.\kigrepair.exe config show --config ".\kigrepair.yaml"
```

Config files must not contain `invite`, tokens, passwords, authorization headers, or secrets.

Reports are written under `C:\ProgramData\kigrepair\Reports\<timestamp>\`. Attach `kigrepair-support-bundle.zip` when escalating.

## Command Safety Classes

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Defender removed Grabber executable

Symptom: `service_binary_missing` or verification reports missing executable.

Likely cause: Defender or AV removed the binary after service registration.

Commands:

```powershell
.\kigrepair.exe check
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Files to check: `initial-detection.json`, `cleanup-plan.json`, `defender-result.json`, `repair-result.json`, `verification-result.json`.

Escalation data: support bundle, Defender status, installer hash.

## Defender status query failed

Symptom: `defender_status_unavailable` or warning in `defender-result.json`.

Likely cause: Defender disabled, not installed, policy restricted, PowerShell cmdlets unavailable.

Commands:

```powershell
.\kigrepair.exe defender
.\kigrepair.exe collect-report
```

Files to check: `defender-result.json`, `operations.json`, `repair.log`.

Escalation data: OS version, security product, policy owner.

## Defender ensure failed

Symptom: `defender --ensure --yes` exits with Defender failure.

Likely cause: Tamper Protection, Group Policy, MDM, or permission failure.

Commands:

```powershell
.\kigrepair.exe defender --ensure --yes
.\kigrepair.exe collect-report
```

Files to check: `defender-result.json`, `operations.json`, `repair.log`.

Escalation data: exact failure message and policy management details.

## Tamper Protection may block changes

Symptom: Defender exclusion add is denied even when elevated.

Likely cause: Microsoft Defender Tamper Protection.

Commands:

```powershell
.\kigrepair.exe defender
.\kigrepair.exe collect-report
```

Files to check: `defender-result.json`, `summary.txt`.

Escalation data: Defender management source and tenant policy notes.

## Group Policy or MDM managed exclusions

Symptom: Exclusions cannot be added or disappear after policy refresh.

Likely cause: Central policy controls Defender settings.

Commands:

```powershell
.\kigrepair.exe defender
.\kigrepair.exe verify
```

Files to check: `defender-result.json`, `verification-result.json`.

Escalation data: GPO/MDM owner and policy name if known.

## Third-party AV detected

Symptom: Defender unavailable or binaries continue disappearing.

Likely cause: Third-party AV/EDR controls protection.

Commands:

```powershell
.\kigrepair.exe check
.\kigrepair.exe collect-report
```

Files to check: `classification-result.json`, diagnostics files, `repair.log`.

Escalation data: AV/EDR product name, policy owner, quarantine evidence.

## MSI validation failed

Symptom: `preflight` or repair dry-run reports installer validation failure.

Likely cause: Missing file, unsupported name, unreadable MSI, invalid metadata, or hash/source issue.

Commands:

```powershell
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Files to check: `installer-validation.json`, `preflight-result.json`.

Escalation data: installer filename, SHA-256, source path.

## Installer missing

Symptom: Preflight says installer is required or auto-discovery failed.

Likely cause: MSI not placed in `assets\` or incorrect path.

Commands:

```powershell
Get-ChildItem .\assets
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Files to check: `preflight-result.json`, `installer-validation.json`.

Escalation data: expected installer name and location.

## Invite missing

Symptom: Preflight or install reports invite missing.

Likely cause: `--invite` omitted or empty.

Commands:

```powershell
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Files to check: `preflight-result.json`.

Escalation data: Do not include the raw invite.

## Admin rights missing

Symptom: Exit code `2` or admin blocker in preflight.

Likely cause: Real mutation requires elevation.

Commands:

```powershell
Start-Process PowerShell -Verb RunAs
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Files to check: `preflight-result.json`, `operations.json`.

Escalation data: Whether UAC was available or RMM ran elevated.

## Service exists but path mismatch

Symptom: Known service exists, but `ImagePath` is unsupported or unexpected.

Likely cause: Manual changes, unsupported install mode, or unrelated service collision.

Commands:

```powershell
.\kigrepair.exe check
.\kigrepair.exe collect-report
```

Files to check: `initial-detection.json`, `classification-result.json`.

Escalation data: Service name and image path from reports.

## Hidden WMI mode inconsistent

Symptom: `wmi_hidden_mode_inconsistent`.

Likely cause: Missing WMI-mode files, unexpected service path, partial cleanup.

Commands:

```powershell
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

Files to check: `verification-result.json`, `initial-detection.json`.

Escalation data: Support bundle required.

## Cleanup plan skips targets

Symptom: `cleanup-plan.json` contains skipped entries.

Likely cause: Path safety validation rejected unknown, malformed, or boundary-risky target.

Commands:

```powershell
.\kigrepair.exe cleanup --dry-run
```

Files to check: `cleanup-plan.json`, `operations.json`.

Escalation data: Skipped target path and reason.

## Repair fails before install

Symptom: Repair exits before MSI install starts.

Likely cause: Preflight blocker, cleanup failure, rollback snapshot failure, or missing confirmation.

Commands:

```powershell
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Files to check: `preflight-result.json`, `repair-plan.json`, `cleanup-result.json`, `rollback-info.json`.

Escalation data: First failed operation from `operations.json`.

## Command failed before making changes

Symptom: A mutating command exits before cleanup, Defender changes, or MSI install begin.

Likely cause: report directory creation failed, confirmation is missing, admin rights are missing, installer validation failed, invite is missing, or rollback snapshot could not be written.

Commands:

```powershell
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Files to check: `summary.txt`, `operations.json`, `preflight-result.json`, `repair-plan.json`, `rollback-info.json`.

Escalation data: Exit code, first failed operation, and report directory path.

## Repair failed before MSI install

Symptom: `repair-result.json` shows `install_executed=false`.

Likely cause: cleanup failed, Defender ensure was required and failed, or rollback snapshot/report writing failed before mutation.

Files to check: `repair-result.json`, `cleanup-result.json`, `defender-result.json`, `operations.json`, `repair.log`.

Escalation data: Whether any cleanup actions executed and whether `rollback-info.json` exists.

## MSI install timed out

Symptom: `msi_install` operation or `install-result.json` reports `failed_timeout` or exit code `-1`.

Likely cause: Windows Installer is blocked, another install is running, installer UI is waiting unexpectedly, or endpoint policy interfered.

Commands:

```powershell
.\kigrepair.exe collect-report
```

Files to check: `install-result.json`, `repair-result.json`, `msi-install.log`, `operations.json`.

Escalation data: MSI log path, timeout status, Windows Installer service state.

## Repair succeeds but verification fails

Symptom: Repair completed but exit code is `7` or `verification_failed`.

Likely cause: Service failed to start, executable missing after install, Defender/AV removed files, or install mode mismatch.

Commands:

```powershell
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

Files to check: `repair-result.json`, `final-detection.json`, `verification-result.json`, `msi-install.log`.

Escalation data: Bundle and installer hash.

## Support bundle generation warning

Symptom: `collect-report` exits with warning.

Likely cause: Some diagnostics were unavailable, event log access failed, or prior report file was locked.

Commands:

```powershell
.\kigrepair.exe collect-report
```

Files to check: `collect-result.json`, `repair.log`.

Escalation data: Attach bundle if created and note skipped collectors.

## Report bundle created with warnings

Symptom: `collect-report` exits `1` and bundle exists.

Likely cause: optional event logs, history files, MSI logs, or report files were missing, locked, inaccessible, or skipped after redaction checks.

Files to check: `collect-result.json`, `operations.json`, `summary.txt`.

Escalation data: Attach the bundle if present and list skipped files from `collect-result.json`.

## Report directory could not be created

Symptom: Command exits before writing normal report files.

Likely cause: `C:\ProgramData\kigrepair\Reports` is inaccessible, the configured report root is invalid, disk is full, or security policy blocks writes.

Commands:

```powershell
Test-Path "C:\ProgramData\kigrepair\Reports"
New-Item -ItemType Directory -Force "C:\ProgramData\kigrepair\Reports"
```

Escalation data: Exact configured report root and the filesystem error. Do not rerun mutating commands until report writing works.

## Report cleanup skipped folder

Symptom: `reports cleanup` leaves an old folder.

Likely cause: Folder outside report root, newest reports protected by retention, lock/permission issue, or safety validation failure.

Commands:

```powershell
.\kigrepair.exe reports cleanup --dry-run
.\kigrepair.exe reports cleanup --yes
```

Files to check: `report-cleanup-plan.json`, `report-cleanup-result.json`.

Escalation data: Skipped folder path and reason.
