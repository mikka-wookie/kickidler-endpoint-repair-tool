# AGENTS.md

## Project name

kigrepair

## Project purpose

`kigrepair` is a Windows-focused Kickidler support utility.

The first major workflow is Grabber detection, repair, cleanup, reinstall, Defender exclusion validation, and reporting.

The app must be designed as an extendable support toolkit, not as a one-time uninstall script replacement.

Future workflows may include:

- log collection
- support bundle generation
- service diagnostics
- network tests
- Defender diagnostics
- registry diagnostics
- MSI install/uninstall analysis
- version verification
- GUI wrapper
- enterprise/RMM-friendly automation

## Primary user

The primary user is a technical support engineer who needs to diagnose and repair Kickidler Grabber installations on Windows endpoints.

The app must reduce repetitive manual reinstall work, especially cases where:

- Windows Defender removed one or more Grabber executables
- service exists but executable is missing
- service is stopped or broken
- registry leftovers remain
- previous uninstall was incomplete
- Defender exclusion is missing
- user cannot manually run cleanup scripts correctly

## Development environment

- OS: Windows
- Editor: VS Code
- Language: Go
- CLI-first application
- GUI may be added later
- PowerShell may be used for Windows-specific detection where appropriate
- Target binary name: `kigrepair.exe`

Use PowerShell examples in documentation and command output unless another shell is explicitly required.

## Build and validation commands

Run these before considering a task complete:

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
```

If a command cannot be run, explain why and provide the exact command the developer should run manually.

## Default report path

All commands must write reports and logs under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Example:

```text
C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\
```

Expected files may include:

```text
initial-detection.json
final-detection.json
operations.json
repair.log
summary.txt
msi-install.log
msi-uninstall.log
kigrepair-support-bundle.zip
```

Only the `reports` package should manage report directory creation and report file paths.

## Architecture principles

The project must remain modular and extendable.

Do not put all logic into `main.go`, Cobra command files, or one large `repair.go`.

Use this responsibility split:

```text
cmd/
  Cobra command entrypoints only.

internal/app/
  Shared context, workflow orchestration, run mode, operation results.

internal/config/
  Constants, service names, paths, process names, registry keys, MSI codes, default paths.

internal/checks/
  Admin check, OS info, architecture, environment checks.

internal/detector/
  Detection only: services, files, processes, registry, Defender, health calculation.

internal/repair/
  High-level repair workflow orchestration.

internal/cleaner/
  Cleanup operations: services, processes, files, registry, MSI uninstall.

internal/installer/
  MSI install, installer location, install verification.

internal/defender/
  Defender status and exclusion operations.

internal/diagnostics/
  Log collection, event logs, system info, process/service snapshots, network diagnostics.

internal/reports/
  Report directory creation, JSON/TXT writing, operations report, archive generation.

internal/safety/
  Path validation, destructive action validation, dry-run safety rules.

internal/winapi/
  Windows-specific wrappers for services, processes, registry, event log, MSI, elevation.

internal/logging/
  File and console logging.

internal/ui/
  Future GUI wrapper. Must call shared workflows instead of duplicating logic.
```

## Windows execution implementation notes

Current MVP may use:

- `sc.exe` for service stop/delete
- `taskkill.exe` for process termination
- PowerShell for process/process path detection and Defender checks
- `msiexec.exe` for MSI install/uninstall
- `reg.exe` for registry export if needed

Rules:

- Keep external command usage isolated in platform/winapi or low-level executor packages.
- Do not scatter `sc.exe`/`taskkill.exe`/PowerShell calls across workflow orchestration code.
- Workflow packages should call internal abstractions, not directly run external commands.
- Long-term target is to replace service/process operations with native Windows APIs where practical.
- `msiexec.exe` usage is acceptable for MSI operations.
- PowerShell usage is acceptable for Defender cmdlets unless a better supported API is added.
- Always log executed command names, arguments where safe, exit codes, stdout/stderr summaries, and errors.
- Never log secrets such as invite values.

## Workflow model

Workflows should follow this concept:

```go
type Workflow interface {
    Name() string
    Run(ctx *AppContext) error
}
```

Examples:

- CheckWorkflow
- RepairWorkflow
- CleanupWorkflow
- InstallWorkflow
- DefenderWorkflow
- CollectReportWorkflow

Every workflow should use shared `AppContext`, logging, reports, and operation results.

## Operation result model

All meaningful actions should return or append a structured operation result.

Expected status values:

```text
success
failed
skipped
warning
```

Every operation result should contain:

- step
- target
- status
- message
- error
- timestamp

Do not silently ignore failed operations.

## Run modes

The app should support these modes:

```text
interactive
cli
quiet
```

Global flags may include:

```text
--output
--quiet
--non-interactive
--force
--json
--no-elevate
```

Rules:

- `--quiet` disables console progress output but still writes logs.
- `--non-interactive` disables prompts and requires required input from flags or auto-detection.
- Quiet/non-interactive mode is for authorized enterprise/RMM usage.
- Quiet mode must still be auditable through logs and reports.
- Interactive admin-required commands may relaunch through normal Windows UAC with `ShellExecute` verb `runas`.
- Quiet or non-interactive mode must not auto-prompt UAC; return the administrator-required exit code instead.
- `--no-elevate` must disable self-elevation and return the administrator-required exit code.
- Do not implement stealth behavior that bypasses security controls or hides from OS auditing.

## CLI commands

Current or planned commands:

```powershell
.\kigrepair.exe check
.\kigrepair.exe repair --invite <INVITE> --installer .\grabber.msi
.\kigrepair.exe cleanup
.\kigrepair.exe install --invite <INVITE> --installer .\grabber.msi
.\kigrepair.exe defender --ensure
.\kigrepair.exe collect-report
.\kigrepair.exe version
```

Do not implement destructive behavior in detection-only tasks.

## Detection logic rules

Detection must be mode-aware.

Grabber may be installed in one of several supported locations:

- standard Program Files path
- helper Program Files path
- hidden WMI path
- ProgramData exact GUID path

Known services:

- `ngs`
- `tls`
- `WmiProviderSE`

Any one of these services may exist. Do not require all services.

Service `ImagePath` is the source of truth when a known service exists.

Detection flow:

1. Detect known services.
2. If one or more services exist, parse service `ImagePath`.
3. Resolve actual service executable path.
4. Derive install root from service executable path.
5. Determine install mode.
6. Check files based on detected mode.
7. Check Defender exclusion based on detected install root.
8. If no services exist, scan known install roots and registry keys.
9. Calculate health.

Do not assume WMI mode unless the active service or discovered files indicate WMI mode.

## Installation modes

Use or preserve this model:

```text
unknown
standard
helper
hidden_wmi
not_installed
```

### Standard mode

Example service executable:

```text
C:\Program Files\TeleLinkSoft\bin\grabber2.exe
```

Required health check:

- service exists
- service is running
- service executable exists
- Defender exclusion covers detected install root

Do not require WMI files in standard mode.

### Helper mode

Example root:

```text
C:\Program Files\TeleLinkSoftHelper
```

Use detected service executable path and root.

### Hidden WMI mode

Example service executable:

```text
C:\Windows\System32\wmi\bin\svchost.exe
```

Expected root:

```text
C:\Windows\System32\wmi
```

Expected binary directory:

```text
C:\Windows\System32\wmi\bin
```

Only in this mode should WMI binaries be treated as required.

## Known cleanup/detection paths

Centralize paths in `internal/config`.

Cleanup paths:

```text
%ProgramFiles%\TeleLinkSoft
%ProgramFiles%\TeleLinkSoftHelper
%ProgramFiles(x86)%\TeleLinkSoft
%ProgramFiles(x86)%\TeleLinkSoftHelper
%ProgramData%\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60
%SystemRoot%\System32\wmi
```

Important exact ProgramData path:

```text
C:\ProgramData\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60
```

Do not use a wildcard GUID pattern unless explicitly requested.

## Known process names

Centralize process names in `internal/config`.

Known process names:

```text
grabber.exe
grabberAgent.exe
grabberSubAgent.exe
grabberSubagent.exe
grabber2.exe
ngsAgent.exe
ngsSubAgent.exe
ngsSubagent.exe
tlshost.exe
tlsservice.exe
tlssubservice.exe
```

Known hidden WMI executable names:

```text
svchost.exe
WmiPrvSE.exe
RuntimeBroker.exe
```

These WMI names are only expected under:

```text
%SystemRoot%\System32\wmi\bin
```

Do not treat normal Windows processes with these names as Grabber processes unless their executable path matches the known hidden WMI path.

## MSI product information

Centralize MSI values in `internal/config`.

MSI product code:

```text
{EB1FBC37-0B97-4CF5-A329-CF28BA653748}
```

MSI packed code:

```text
73CBF1BE79B05FC43A92FC82AB567384
```

Silent install command shape:

```powershell
msiexec /i "<installer>" /qn /norestart invite=<INVITE> /l*v "<output>\msi-install.log"
```

Silent uninstall command shape:

```powershell
msiexec /x {EB1FBC37-0B97-4CF5-A329-CF28BA653748} /qn /norestart /l*v "<output>\msi-uninstall.log"
```

## Registry keys

Known product keys:

```text
HKCU\Software\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper
HKLM\SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper
HKLM\SOFTWARE\WOW6432Node\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper
```

Known MSI keys:

```text
HKCR\Installer\Features\73CBF1BE79B05FC43A92FC82AB567384
HKCR\Installer\Products\73CBF1BE79B05FC43A92FC82AB567384
HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}
HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}
HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products\73CBF1BE79B05FC43A92FC82AB567384
```

Registry detection is independent from file/service detection.

Registry leftovers alone may indicate `partially_removed`, but should not force WMI file checks.

## Defender detection rules

Defender exclusion detection must be path-normalized.

Do not parse human/table-formatted PowerShell output.

Prefer JSON output from PowerShell:

```powershell
$p=(Get-MpPreference).ExclusionPath; if ($null -eq $p) { @() | ConvertTo-Json -Compress } else { @($p) | ConvertTo-Json -Compress }
```

Normalize paths before comparison:

- trim whitespace
- trim quotes
- expand environment variables
- replace `/` with `\`
- clean path
- remove trailing slash except drive root
- compare case-insensitively
- handle `\\?\` prefix

A Defender exclusion covers a required path if:

- paths are exact match, or
- required path is inside the excluded parent path

Examples:

```text
Required:  C:\Program Files\TeleLinkSoft\bin
Exclusion: C:\Program Files\TeleLinkSoft
Result:    covered
```

Boundary protection:

```text
Required:  C:\Program Files\TeleLinkSoft2
Exclusion: C:\Program Files\TeleLinkSoft
Result:    not covered
```

Defender required paths must be based on detected install root.

Do not require WMI Defender exclusion for a standard Program Files installation.

If Defender detection is unavailable:

- do not fail the whole check
- health can still be healthy if service/files are healthy
- add warning issue: Defender exclusions could not be verified

## Health calculation rules

Health statuses:

```text
healthy
broken
partially_removed
not_installed
unknown
```

### Healthy

- at least one known service exists
- selected primary service is running
- primary service image path is usable
- service executable exists
- install root is detected
- Defender exclusion covers detected install root, or Defender detection is unavailable with warning

### Broken

- known service exists but service executable is missing
- known service exists but install root cannot be resolved
- primary service is stopped and executable is missing
- hidden WMI mode is detected but required WMI executable is missing

### Partially removed

- no known service exists, but known install folders contain files
- no known service exists, but registry keys exist
- known folders exist but no known executable exists
- ProgramData exact folder exists without service

### Not installed

- no known services
- no known folders/files
- no known registry keys

### Unknown

- major detection failures prevent classification

Do not mark a standard Program Files installation as broken because WMI files are missing.

## Path safety rules

Before implementing file deletion, validate paths.

Allowed cleanup paths must be exact expanded paths from config.

Reject:

- empty path
- relative path
- unresolved environment variables
- drive root
- Windows root
- System32 root
- unknown path
- malformed path

Destructive file deletion must never run without path validation.

## Destructive action policy

For tasks involving cleanup, registry deletion, service deletion, process killing, Defender changes, or MSI install/uninstall:

- Ask for explicit confirmation if the user did not clearly request the destructive action.
- Prefer implementing dry-run first.
- Log every action.
- Return structured operation results.
- Write before/after reports.
- Verify final state.
- Use stable exit codes.

Detection-only tasks must not modify the system.

## Testing expectations

Add tests for:

- health calculation
- install mode detection
- service ImagePath parsing
- path normalization
- Defender exclusion coverage
- cleanup path validation
- registry key parsing if applicable

Important Defender test cases:

- exact match
- trailing slash
- case-insensitive match
- parent covers child
- child does not cover parent
- boundary protection
- WMI parent covers bin
- whitespace/quotes
- forward slash normalization

## Console output expectations

For `check`, output should be readable:

```text
Kigrepair Check Summary

Health: healthy
Install mode: standard
Install root: C:\Program Files\TeleLinkSoft\bin
Primary service: ngs
Service status: running
Service executable: C:\Program Files\TeleLinkSoft\bin\grabber2.exe
Defender exclusion: present

Issues:
- ...

Recommendations:
- ...

Report:
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

If `--json` is used:

- print JSON to stdout
- still write files to report directory

If `--quiet` is used:

- do not print progress
- still write logs and reports

## Task discipline

Work in small steps.

Preferred implementation order:

1. project structure
2. detection
3. install mode correction
4. Defender exclusion comparison fix
5. dry-run cleanup
6. real cleanup
7. install/reinstall
8. verification
9. collect-report/support bundle
10. self-elevation/UAC
11. GUI

Do not implement multiple future features in one task unless explicitly requested.

## Response format after changes

After making changes, summarize:

```text
What changed
Files modified
Validation run
Risks/limitations
Next recommended step
```

Mention additional findings separately instead of changing unrelated logic without approval.

Git recommendation text is allowed when it is relevant, for example suggesting that the developer review, stage, or commit completed changes. Do not run Git write operations unless explicitly requested.
