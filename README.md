# kigrepair

`kigrepair` is a Windows-focused Kickidler support utility for detecting, cleaning, installing, and later repairing Grabber installations.

The project is CLI-first and intended for support engineers who need auditable diagnostics and recovery actions on Windows endpoints. It is designed as an extendable support toolkit, not a one-time cleanup script.

## Current Commands

Implemented commands:

- `check` - detects the current Grabber installation state, classifies health, checks files/services/registry/Defender state, and writes reports.
- `cleanup --dry-run` - builds and writes a cleanup plan without modifying the system.
- `cleanup` - performs real cleanup of planned services, processes, files, registry leftovers, and MSI uninstall steps after confirmation and safety validation.
- `install` - runs the MSI install command with the provided invite and installer path, then verifies final state.
- `defender` - reports Defender exclusion status for the detected installation.
- `defender --ensure` - adds missing Defender exclusions after confirmation.
- `version` - prints the application version.

Related placeholder/planned commands may exist in the CLI, but the full repair workflow is not implemented yet.

Global flags:

- `--output string` - report output directory or report root.
- `--quiet` - suppress console output.
- `--non-interactive` - disable prompts.
- `--force` - reserved for privileged workflows.
- `--json` - print command results as JSON.

## Current Workflow Status

Implemented:

- detection/check workflow
- mode-aware install path detection
- Defender exclusion detection
- cleanup dry-run
- real cleanup
- MSI install command
- Defender status/ensure command

Not implemented yet:

- full repair workflow
- GUI
- collect-report/support bundle improvements if not complete
- native Windows service/process APIs

## Reports

All reports and logs are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Example:

```text
C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\
```

Common files:

- `initial-detection.json`
- `final-detection.json`
- `operations.json`
- `summary.txt`
- `repair.log`
- `cleanup-plan.json`
- `install-result.json`
- `defender-result.json`
- `msi-install.log`
- `msi-uninstall.log`

Quiet and non-interactive modes still write logs and reports.

## Command Examples

Check:

```powershell
.\kigrepair.exe check
```

Cleanup dry-run:

```powershell
.\kigrepair.exe cleanup --dry-run
```

Real cleanup interactive:

```powershell
.\kigrepair.exe cleanup
```

Real cleanup automation:

```powershell
.\kigrepair.exe cleanup --yes --quiet --non-interactive
```

Install:

```powershell
.\kigrepair.exe install --invite <INVITE> --installer .\grabber.msi
```

Defender status:

```powershell
.\kigrepair.exe defender
```

Defender ensure:

```powershell
.\kigrepair.exe defender --ensure
```

Defender ensure automation:

```powershell
.\kigrepair.exe defender --ensure --yes --quiet --non-interactive
```

Version:

```powershell
.\kigrepair.exe version
```

## Safety Notes

- `check` does not modify the system.
- `cleanup --dry-run` does not modify the system.
- Real cleanup requires administrator rights.
- Install requires administrator rights.
- `defender --ensure` requires administrator rights.
- Quiet/non-interactive modes still write logs and reports.
- Invite values must not be logged.
- Real cleanup and Defender ensure are intended for authorized support or enterprise automation use only.

## Cleanup Safety

Real cleanup is destructive and requires confirmation unless automation flags are provided:

```powershell
.\kigrepair.exe cleanup --yes --quiet --non-interactive
```

Cleanup actions are produced by the cleanup planner and safety validation. Detection-only commands must not perform cleanup, repair, reinstall, Defender changes, or MSI actions.

Current cleanup action order:

1. `stop_service`
2. `kill_process`
3. `msi_uninstall`
4. `delete_service`
5. `delete_path`
6. `delete_registry_key`

## MVP Implementation Notes

- Service operations currently may use `sc.exe`.
- Process termination currently may use `taskkill.exe`.
- Process and Defender detection may use PowerShell.
- MSI operations use `msiexec.exe`.
- This is acceptable for the MVP.
- Long-term target is to isolate or replace service/process operations with native Windows APIs where practical.

External command usage should remain isolated in low-level packages or Windows-specific wrappers. Cobra command entrypoints and workflow orchestration should call internal abstractions rather than running system commands directly.

## Next Planned Step

Full repair workflow:

```text
detect -> cleanup -> defender ensure -> install -> final verification -> report
```

Do not implement this step yet.

## Development Setup

Requirements:

- Windows
- Go 1.22 or newer
- VS Code with the Go extension

Build:

```powershell
go build -o kigrepair.exe ./cmd/kigrepair
```

Validate:

```powershell
gofmt -w .
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
```

To write reports to a local directory during development:

```powershell
.\kigrepair.exe check --output .\Reports\dev-check
```

## Architecture

The project is split into internal packages for app context, workflows, configuration, detection, cleanup, installation, Defender integration, diagnostics, reports, safety validation, Windows-specific wrappers, logging, and future UI surfaces.

Keep workflow orchestration separate from low-level Windows operations. Destructive actions should go through planning, safety validation, operation result recording, logging, before/after detection, and report writing.
