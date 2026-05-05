# kigrepair

`kigrepair` is a Windows-focused Kickidler support utility for Grabber diagnostics, cleanup, install, repair, Defender exclusion validation, verification, and support bundle collection.

The tool is CLI-first and intended for technical support engineers. It writes auditable reports under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

## Supported OS

- Windows endpoints
- Go development target: Windows amd64 release builds

System-changing commands require administrator rights. Interactive mode can request standard Windows UAC elevation. Quiet or non-interactive automation must already run from an elevated shell.

## Safety

Read-only/non-destructive commands, except for report writing:

- `check`
- `verify`
- `preflight`
- `repair --dry-run`
- `cleanup --dry-run`
- `collect-report`
- `reports list`

System-changing commands:

- `cleanup`
- `repair`
- `defender --ensure`
- `install`
- `reports cleanup --yes`

Cleanup, repair, Defender changes, MSI install/uninstall, process termination, service changes, registry cleanup, and report cleanup are logged and reported. Invite values must not be printed, logged, committed, or included in tickets. Command examples use `<INVITE>`.

## Common Commands

```powershell
.\kigrepair.exe version
.\kigrepair.exe version --json
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe cleanup --dry-run
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
.\kigrepair.exe collect-report
.\kigrepair.exe reports list
.\kigrepair.exe reports cleanup --dry-run
```

## Reports

Each workflow command except `version` writes a timestamped report directory. Common files include:

- `summary.txt`
- `operations.json`
- `repair.log`
- `initial-detection.json`
- `final-detection.json`
- `verification-result.json`
- `preflight-result.json`
- `repair-result.json`
- `collect-result.json`
- `kigrepair-support-bundle.zip`

`summary.txt`, key workflow result JSON files, and support bundle manifests include build metadata so support can identify the exact binary used.

## MSI Assets

Release packages include an empty `assets\` folder with instructions. Place a supported Grabber MSI there when using installer auto-discovery:

- `grabberEM.x64.msi`
- `grabberEM.x32.msi`
- `grabberTT.x64.msi`
- `grabberTT.x32.msi`
- `grabber.msi`

Do not place invite values in files.

## Support Bundles

Run:

```powershell
.\kigrepair.exe collect-report
```

The support bundle is written to the active report directory as:

```text
kigrepair-support-bundle.zip
```

Attach this ZIP to escalation tickets. Do not include invite values in ticket notes.

## Exit Codes

Standard workflow exit codes:

- `0` - success
- `1` - completed with warnings
- `2` - administrator rights required
- `3` - confirmation declined or missing
- `4` - invalid input
- `5` - cleanup failed
- `6` - install failed
- `7` - final verification failed
- `8` - Defender ensure failed
- `9` - reboot required
- `10` - unexpected error

`check` health-oriented exit codes:

- `0` - healthy
- `1` - not installed
- `2` - broken
- `3` - partially removed
- `4` - unknown

## Release Build

From the repository root:

```powershell
.\scripts\build-release.ps1 -Version 0.1.0
```

If local PowerShell execution policy blocks direct script execution, run:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-release.ps1 -Version 0.1.0
```

The script runs `gofmt -w .`, `go mod tidy`, `go test ./...`, and a Windows amd64 build with linker-injected metadata. It creates:

```text
dist\kigrepair-0.1.0-windows-amd64\
dist\kigrepair-0.1.0-windows-amd64.zip
```

Release folder layout:

```text
kigrepair.exe
README.md
SUPPORT-RUNBOOK.md
checksums.txt
checksums.json
assets\README.txt
examples\commands.ps1
```

The release script does not bundle Grabber MSI installers. It creates an `assets\` folder for support engineers to populate locally when needed.

## Development Validation

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
```

## Architecture

The project is split into internal packages for app context, workflows, configuration, detection, cleanup, installation, Defender integration, diagnostics, reports, safety validation, Windows wrappers, logging, and future UI surfaces. Keep workflow orchestration separate from low-level Windows operations.
