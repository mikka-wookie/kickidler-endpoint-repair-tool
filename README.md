# kigrepair

`kigrepair` is a Windows CLI-first support utility for Kickidler support engineers. The current implemented workflows focus on Grabber detection, health classification, cleanup planning, real cleanup execution, Defender exclusion detection, and report generation.

The project is designed as an extendable support toolkit. Repair/reinstall, Defender modification, MSI install, support bundles, service diagnostics, network tests, and GUI integration are planned future areas.

## Commands

- `check` - detect Grabber installation state and write a report.
- `cleanup --dry-run` - generate and report a cleanup plan without changing the system.
- `cleanup` - execute real cleanup after confirmation.
- `repair` - placeholder workflow; reinstall/repair is not implemented yet.
- `install` - placeholder workflow; MSI install is not implemented yet.
- `defender` - placeholder workflow; Defender settings are not modified yet.
- `collect-report` - placeholder diagnostic collection workflow.
- `version` - print the application version.

Global flags:

- `--output string`
- `--quiet`
- `--non-interactive`
- `--force`
- `--json`

Cleanup flags:

- `--dry-run` - preview cleanup actions only.
- `--yes` - confirm real cleanup without prompting.

Command-specific placeholder flags:

- `repair --invite string --installer string`
- `install --invite string --installer string`
- `defender --ensure`

## Cleanup Safety

Dry-run cleanup is non-destructive:

```powershell
.\kigrepair.exe cleanup --dry-run
```

Real cleanup is destructive and requires administrator rights:

```powershell
.\kigrepair.exe cleanup
```

In interactive mode, real cleanup prints the planned actions and requires the user to type exactly:

```text
YES
```

Automation mode must provide `--yes`:

```powershell
.\kigrepair.exe cleanup --yes --quiet --non-interactive
```

If `--quiet` or `--non-interactive` is used without `--yes`, cleanup exits with code `3` and does not run destructive actions.

Real cleanup only executes actions produced by the cleanup planner and marked safe. It does not perform repair, reinstall, Defender exclusion changes, or MSI install.

Execution order:

1. `stop_service`
2. `kill_process`
3. `msi_uninstall`
4. `delete_service`
5. `delete_path`
6. `delete_registry_key`

## Exit Codes

Cleanup uses stable exit codes:

- `0` - cleanup completed successfully
- `1` - cleanup completed with warnings
- `2` - administrator rights required
- `3` - confirmation declined or missing
- `4` - cleanup blocked by safety validation
- `5` - cleanup failed partially
- `10` - unexpected error

## Reports

By default, reports are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Example:

```text
C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\
```

Cleanup reports may include:

- `initial-detection.json`
- `cleanup-plan.json`
- `operations.json`
- `summary.txt`
- `repair.log`
- `final-detection.json`
- `msi-uninstall.log`
- `registry-backup\*.reg`

## Windows Execution MVP

The current MVP uses external Windows tools in low-level executor/detector packages:

- `sc.exe` for service query, stop, and delete
- `taskkill.exe` for process termination by PID
- PowerShell/CIM for process detection and process path verification
- PowerShell Defender cmdlets for Defender exclusion detection
- `msiexec.exe` for MSI uninstall
- `reg.exe` for registry export backup

These calls should remain isolated behind low-level packages such as `internal/cleaner`, `internal/detector`, or future `internal/winapi` wrappers. Cobra commands and workflow orchestration should call internal abstractions, not run external commands directly.

Long-term, service and process operations should move to native Windows APIs where practical. `msiexec.exe` remains acceptable for MSI operations, and PowerShell remains acceptable for Defender cmdlets unless a better supported API is added.

## Development Setup

Requirements:

- Windows
- Go 1.22 or newer
- VS Code with the Go extension

Install dependencies:

```powershell
go mod tidy
```

Build:

```powershell
go build -o kigrepair.exe ./cmd/kigrepair
```

Validate:

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
```

Run examples:

```powershell
.\kigrepair.exe check
.\kigrepair.exe cleanup --dry-run
.\kigrepair.exe cleanup
.\kigrepair.exe cleanup --yes --quiet --non-interactive
.\kigrepair.exe collect-report --json
.\kigrepair.exe version
```

To write reports to a local directory during development:

```powershell
.\kigrepair.exe check --output .\Reports\dev-check
```

## Architecture

The project is split into internal packages for app context, workflows, configuration, detection, repair, cleanup, installation, Defender integration, diagnostics, reports, safety validation, Windows-specific wrappers, logging, and future UI surfaces.

Destructive cleanup actions must go through cleanup planning, safety validation, operation result recording, logging, and before/after reporting.
