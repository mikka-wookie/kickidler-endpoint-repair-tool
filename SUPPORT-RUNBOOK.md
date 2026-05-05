# kigrepair Support Runbook

Use this runbook from an elevated PowerShell when running system-changing commands. Read-only commands can run without elevation, but some checks may report warnings if Windows APIs are not readable.

## 1. Basic Triage

```powershell
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Review `summary.txt` in the report directory:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

## 2. Safe Preview

```powershell
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Dry-run repair writes the planned actions and does not change services, processes, files, registry, Defender, or MSI state.

## 3. Real Repair

```powershell
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Real repair can stop/delete services, terminate supported Grabber processes, remove allowlisted files and registry keys, run MSI install/uninstall, and ensure Defender exclusions.

## 4. Collect Bundle

```powershell
.\kigrepair.exe collect-report
```

Attach `kigrepair-support-bundle.zip` from the report directory to the escalation ticket.

## 5. Report Cleanup

```powershell
.\kigrepair.exe reports cleanup --dry-run
.\kigrepair.exe reports cleanup --yes
```

Use dry-run first to confirm which old report folders will be removed.

## 6. Common Classifications

- `healthy` - Grabber appears installed and operational. Review warnings, then no repair is usually required.
- `not_installed` - No known service, files, or registry keys were found. Install may be appropriate if the endpoint should have Grabber.
- `service_binary_missing` - A known service exists but its executable is missing. Run repair with a valid installer.
- `defender_exclusion_missing` - Defender exclusion coverage is missing for the detected install root. Run repair or `defender --ensure`.
- `partial_msi_leftovers` - MSI registry leftovers remain. Run repair dry-run, then real repair if approved.
- `partial_files_leftover` - Known files/folders remain without a healthy service. Run cleanup dry-run, then repair if reinstall is needed.
- `wmi_hidden_mode_inconsistent` - Hidden WMI mode evidence is inconsistent. Escalate with a support bundle if repair does not restore a healthy state.
- `unknown_install_state` - Detection could not classify the endpoint. Collect a support bundle and escalate.

## 7. Escalation Checklist

- Attach `kigrepair-support-bundle.zip`.
- Include customer symptoms.
- Include installer filename and SHA-256 hash if relevant.
- Include the exact command used, replacing the invite with `<INVITE>`.
- Do not include the invite value in ticket notes.
