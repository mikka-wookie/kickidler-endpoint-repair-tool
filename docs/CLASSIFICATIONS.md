# kigrepair Classifications

Classification codes are support-facing summaries used to choose the next action. They do not replace reading the report files.

## Command Safety Classes

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## healthy

Meaning: Grabber appears installed and operational.

Typical evidence: Known service exists and is running, executable exists, install root is detected, verification hard requirements passed.

Severity: Info.

Recommended next action: No repair required. Collect a bundle only if the customer symptom continues.

Escalation requirement: Escalate if user symptoms persist despite healthy verification.

## not_installed

Meaning: No known services, files, or registry keys were found.

Typical evidence: Detection found no supported Grabber artifacts.

Severity: Info or warning depending customer expectation.

Recommended next action:

```powershell
.\kigrepair.exe install --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Escalation requirement: Escalate only if install is expected but blocked.

## service_binary_missing

Meaning: A supported Grabber service exists and its `ImagePath` points to a known Grabber executable, but the executable file is missing.

Typical evidence: Service exists, service path resolves, file check fails.

Severity: Critical.

Recommended next action:

```powershell
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Escalation requirement: Escalate if repair completes and verification still fails.

## service_not_running

Meaning: A supported service exists but is stopped.

Typical evidence: Service status is stopped while executable exists.

Severity: Warning.

Recommended next action: Run `verify`, then `repair --dry-run`; run real repair if the plan is valid.

Escalation requirement: Escalate if service cannot be started or repeatedly stops.

## defender_exclusion_missing

Meaning: Defender exclusion coverage is missing for the detected install root.

Typical evidence: Defender path comparison reports not covered.

Severity: Warning or critical if files were removed.

Recommended next action:

```powershell
.\kigrepair.exe defender --ensure --yes
```

Escalation requirement: Escalate if policy, MDM, or Tamper Protection blocks changes.

## partial_msi_leftovers

Meaning: MSI registry leftovers exist without a healthy installation.

Typical evidence: Known MSI product or installer registry keys exist while service/files are missing.

Severity: Warning.

Recommended next action: Run `repair --dry-run`, review `cleanup-plan.json`, then repair with `--yes` if approved.

Escalation requirement: Escalate if MSI validation or uninstall repeatedly fails.

## partial_files_leftover

Meaning: Known Grabber folders or files remain without a healthy service.

Typical evidence: Known cleanup paths contain files and no supported service is active.

Severity: Warning.

Recommended next action:

```powershell
.\kigrepair.exe cleanup --dry-run
```

Then run real cleanup or repair only after reviewing the plan.

Escalation requirement: Escalate if skipped targets are unexpected or path validation blocks cleanup.

## wmi_hidden_mode_inconsistent

Meaning: Hidden WMI mode evidence is incomplete or inconsistent.

Typical evidence: WMI service/path indicators do not agree, or expected WMI binaries are missing in hidden WMI mode.

Severity: Critical.

Recommended next action: Collect a support bundle and run repair only if the plan clearly targets supported paths.

Escalation requirement: Escalate if still present after one repair attempt.

## unknown_install_state

Meaning: Detection cannot confidently classify the endpoint.

Typical evidence: Detection failures, unsupported paths, path mismatch, or conflicting artifacts.

Severity: Critical.

Recommended next action:

```powershell
.\kigrepair.exe collect-report
```

Escalation requirement: Required.

## verification_failed

Meaning: Final or standalone verification failed a hard requirement.

Typical evidence: Missing executable, missing service, unresolved install root, failed final detection.

Severity: Critical.

Recommended next action: Review `verification-result.json`, `final-detection.json`, and `operations.json`; collect a bundle.

Escalation requirement: Required after repair.

## verification_warning

Meaning: Verification passed hard requirements but warnings remain.

Typical evidence: Defender unavailable, process warning, policy warning, or non-critical diagnostic warning.

Severity: Warning.

Recommended next action: Review warnings and run `collect-report` if customer symptoms persist.

Escalation requirement: Escalate if warnings map to policy or environment blockers.

## defender_status_unavailable

Meaning: Defender status or exclusions could not be read.

Typical evidence: PowerShell Defender cmdlet failed, Defender absent/disabled, policy blocks access.

Severity: Warning.

Recommended next action: Check Defender/AV management status and collect a bundle.

Escalation requirement: Escalate if support needs to prove exclusion coverage.

## defender_unavailable_or_disabled

Meaning: Defender is not available or appears disabled.

Typical evidence: Defender service or cmdlets unavailable.

Severity: Warning.

Recommended next action: Confirm whether third-party AV or policy is expected.

Escalation requirement: Escalate if endpoint policy is unknown.

## third_party_av_detected

Meaning: A third-party antivirus or EDR product may control exclusions or remediation.

Typical evidence: Defender disabled/unavailable, security product evidence in diagnostics, or policy-managed exclusions.

Severity: Warning.

Recommended next action: Follow customer AV/EDR exclusion process and collect bundle data.

Escalation requirement: Escalate if AV/EDR continues removing binaries.
