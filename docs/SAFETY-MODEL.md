# kigrepair Safety Model

`kigrepair` must never delete, stop, kill, or modify anything based only on a loose name match. Mutating workflows operate on validated, allowlisted targets and write auditable report files.

Reports are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

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

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Read-only workflows

Read-only workflows may write reports, logs, and support bundles, but they do not modify services, processes, files, registry, Defender exclusions, MSI state, or report retention state.

Examples:

```powershell
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe cleanup --dry-run
.\kigrepair.exe collect-report
```

## Mutating workflows

Mutating workflows require approval and write before/after reports where applicable.

Examples:

```powershell
.\kigrepair.exe cleanup --yes
.\kigrepair.exe install --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
.\kigrepair.exe defender --ensure --yes
.\kigrepair.exe reports cleanup --yes
```

## Admin requirements

System-modifying commands require administrator rights. Quiet, non-interactive, or `--no-elevate` execution must not trigger a UAC prompt; it should return an administrator-required exit code instead.

## `--yes` requirements

`--yes` is required for approved non-interactive mutation. Missing confirmation should stop the operation before mutation.

## Cleanup plan validation

Cleanup targets come from centralized configuration and detection. Paths must be absolute, expanded, allowlisted, and protected against drive root, Windows root, System32 root, unknown paths, unresolved variables, malformed paths, and boundary mistakes.

## Real cleanup executes only validated plan

Real cleanup executes only the validated cleanup plan. It must not invent new deletion, service, process, registry, or MSI targets during execution.

## Service mutation safety

Service stop/delete operations require a supported service name and trusted path evidence. Unknown services must not be changed.

## Process termination safety

Process termination requires trusted PID/name/path evidence and a cleanup plan entry. Hidden WMI executable names such as `svchost.exe`, `WmiPrvSE.exe`, and `RuntimeBroker.exe` are considered Grabber-related only under the supported hidden WMI path.

## Defender separation

Defender status reads configuration. Defender ensure changes exclusions and must require confirmation and administrator rights.

## MSI validation

MSI validation runs before destructive repair or install. Invalid, missing, unreadable, or unsupported installers should stop the workflow before mutation.

## Rollback and audit

Rollback/change snapshot information should be written before mutation where practical. `rollback-info.json`, `operations.json`, and `repair.log` are audit artifacts.

## Report cleanup boundary

Report cleanup is limited to validated timestamped report directories under the configured report root. It must not delete arbitrary folders.

## Support bundle

`collect-report` writes `kigrepair-support-bundle.zip`. The bundle is for escalation and should not contain raw invite values.
