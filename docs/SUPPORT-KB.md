# kigrepair Support KB

## Purpose

`kigrepair` is a Windows support utility for diagnosing and repairing Kickidler Grabber installations. It is intended for technical support engineers and writes reports under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

## Supported use cases

- Grabber service exists but executable is missing.
- Defender exclusion is missing or cannot be verified.
- MSI leftovers remain after uninstall.
- Program Files or ProgramData leftovers remain.
- Hidden WMI mode is inconsistent.
- Verification is needed after repair.
- Diagnostic bundle collection is needed for escalation.

## Do not use for

- Non-Windows machines.
- Unrelated Windows services or third-party software.
- Manual deletion outside the generated cleanup plan.
- Unknown third-party tools.
- Bypassing Defender, EDR, policy, or OS auditing.

## Command Safety Classes

Read-only / non-destructive except report writing:

- `check`
- `verify`
- `preflight`
- `repair --dry-run`
- `cleanup --dry-run`
- `collect-report`
- `reports list`
- `reports cleanup --dry-run`
- `version`

System-modifying / destructive:

- `cleanup --yes`
- `repair --yes`
- `install --yes`
- `defender ensure --yes` using `.\kigrepair.exe defender --ensure --yes`
- `reports cleanup --yes`

## Invite Safety

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Safety model

Mutating workflows require explicit confirmation with `--yes` or an interactive prompt. Cleanup and repair operate from validated plans and write auditable files. `kigrepair` must never delete, stop, kill, or modify anything based only on a loose name match.

## Quick triage

```powershell
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

Review `summary.txt`, `classification-result.json`, and `verification-result.json` in the report directory.

## Standard pre-repair flow

```powershell
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Confirm that `preflight-result.json` is ready or ready with warnings, and inspect `repair-plan.json` before running a real repair.

## Real repair

Destructive: this can stop/delete supported services, terminate trusted Grabber processes, remove allowlisted files and registry keys, run MSI uninstall/install, and add Defender exclusions.

```powershell
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

## After repair

```powershell
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

The expected support bundle name is:

```text
kigrepair-support-bundle.zip
```

## Escalation

Escalate when classification is `unknown_install_state`, `wmi_hidden_mode_inconsistent`, repeated `verification_failed`, service path mismatch, process path mismatch, invalid official MSI, rollback snapshot failure, or repair modifies nothing while state remains broken.

Attach `kigrepair-support-bundle.zip` and include the exact command used with the invite replaced by `<INVITE>`.

## Common mistakes

- Running real repair before `preflight` and `repair --dry-run`.
- Using a real invite in a ticket or screenshot.
- Running from a non-elevated shell for system-modifying commands.
- Treating Defender status warnings as proof that exclusions were changed.
- Manually deleting folders that were not in `cleanup-plan.json`.
- Escalating without the support bundle.
