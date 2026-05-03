# kigrepair

`kigrepair` is a Windows CLI-first support and repair utility for Kickidler support engineers. The first planned workflow is Grabber repair/reinstall, with room for later diagnostics, log collection, support bundles, service checks, network tests, and GUI integration.

Current status: skeleton only. This version does not delete files, modify the registry, stop services, kill processes, run installers, uninstall MSI products, or add Microsoft Defender exclusions.

## Commands

- `check` - run placeholder detection and health checks.
- `repair` - run placeholder Grabber repair workflow.
- `cleanup` - write a placeholder cleanup plan.
- `install` - run placeholder Grabber install workflow.
- `defender` - inspect placeholder Defender workflow.
- `collect-report` - run placeholder diagnostic collectors.
- `version` - print the application version.

Global flags:

- `--output string`
- `--quiet`
- `--non-interactive`
- `--force`
- `--json`

Command-specific flags:

- `repair --invite string --installer string`
- `install --invite string --installer string`
- `defender --ensure`

## Reports

By default, reports are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Example:

```text
C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\
```

Each run creates placeholder report files, `operations.json`, and `repair.log`.

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

Run examples:

```powershell
.\kigrepair.exe check
.\kigrepair.exe repair --invite "INVITE_PLACEHOLDER" --installer "C:\Temp\grabber.msi"
.\kigrepair.exe install --invite "INVITE_PLACEHOLDER"
.\kigrepair.exe defender --ensure
.\kigrepair.exe collect-report --json
.\kigrepair.exe version
```

To write reports to a local directory during development:

```powershell
.\kigrepair.exe check --output .\Reports\dev-check
```

## Architecture

The project is split into internal packages for app context, workflows, configuration, detection, repair, cleanup, installation, Defender integration, diagnostics, reports, safety validation, Windows API adapters, logging, and future UI surfaces.

Real destructive actions are intentionally absent. Future implementation should route privileged operations through `internal/safety` validation and write a clear operation result for every step.
