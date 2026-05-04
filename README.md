# kigrepair

`kigrepair` is a Windows-focused Kickidler support utility for detecting, cleaning, installing, repairing, and collecting diagnostics for Grabber installations.

The project is CLI-first and intended for support engineers who need auditable diagnostics and recovery actions on Windows endpoints. It is designed as an extendable support toolkit, not a one-time cleanup script.

## Commands

Implemented commands:

- `check` - detects the current Grabber installation state, classifies health, checks files/services/registry/Defender state, and writes reports.
- `cleanup --dry-run` - builds and writes a cleanup plan without modifying the system.
- `cleanup` - performs real cleanup of planned services, processes, files, registry leftovers, and MSI uninstall steps after confirmation and safety validation.
- `install` - resolves a Grabber MSI, runs the MSI install command with the provided invite, then verifies final state.
- `defender` - reports Defender exclusion status for the detected installation.
- `defender --ensure` - adds missing Defender exclusions after confirmation.
- `repair` - runs detection, cleanup when needed, MSI install when needed, Defender ensure when needed, final verification, and reports.
- `collect-report` - collects read-only diagnostics and creates a support bundle unless `--no-zip` is used.
- `version` - prints the application version and does not create a report directory.

Global flags:

- `--output string` - report output directory.
- `--quiet` - suppress human-readable console output.
- `--non-interactive` - disable prompts.
- `--force` - allow repair to reinstall even when the detected state is healthy.
- `--json` - print one JSON object to stdout.
- `--no-elevate` - return an administrator-required error instead of relaunching through UAC.

## Reports

Every command except `version` creates one report directory per run:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Example:

```text
C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\
```

Common files:

- `operations.json`
- `summary.txt`
- `repair.log`

Command-specific files:

- `check`: `initial-detection.json`
- `cleanup --dry-run`: `initial-detection.json`, `cleanup-plan.json`
- `cleanup`: `initial-detection.json`, `cleanup-plan.json`, `final-detection.json`, `msi-uninstall.log` when MSI uninstall runs
- `install`: `initial-detection.json`, `final-detection.json`, `install-result.json`, `msi-install.log`
- `defender`: `initial-detection.json`, `defender-result.json`, and `final-detection.json` when `--ensure` verifies final state
- `repair`: `initial-detection.json`, `final-detection.json`, `repair-result.json`, `cleanup-plan.json`, `install-result.json` when install runs, `defender-result.json` when Defender ensure runs, MSI logs when relevant
- `collect-report`: `detection.json`, `system.json`, `services.json`, `processes.json`, `defender.json`, `registry.json`, `collect-result.json`, and `kigrepair-support-bundle.zip` unless `--no-zip` is used

`operations.json` is a JSON array of operation results. Each operation contains `step`, `target`, `status`, `message`, optional `error`, and `timestamp`.

## Examples

Check:

```powershell
.\kigrepair.exe check
.\kigrepair.exe check --json
```

Cleanup dry-run:

```powershell
.\kigrepair.exe cleanup --dry-run
```

Real cleanup:

```powershell
.\kigrepair.exe cleanup
.\kigrepair.exe cleanup --yes --quiet --non-interactive
```

Install:

```powershell
.\kigrepair.exe install --invite <INVITE>
.\kigrepair.exe install --invite <INVITE> --installer .\grabberTT.x64.msi
```

Defender:

```powershell
.\kigrepair.exe defender
.\kigrepair.exe defender --ensure
.\kigrepair.exe defender --ensure --yes --quiet --non-interactive
```

Repair:

```powershell
.\kigrepair.exe repair --invite <INVITE> --yes
.\kigrepair.exe repair --invite <INVITE> --yes --quiet --non-interactive
.\kigrepair.exe repair --invite <INVITE> --installer .\grabberTT.x64.msi --yes
```

Collect diagnostics:

```powershell
.\kigrepair.exe collect-report
.\kigrepair.exe collect-report --no-zip
```

Version:

```powershell
.\kigrepair.exe version
.\kigrepair.exe version --json
```

Support release packages are expected to contain `kigrepair.exe` and may also contain one or more supported Grabber MSI installers. When a supported MSI is included near the executable, support engineers can run:

```powershell
.\kigrepair.exe repair --invite <INVITE> --yes
```

## Installer Auto-Discovery

`install` and `repair` can run without `--installer`. `install` always requires a resolved MSI. `repair` resolves the MSI only when the repair decision requires an install, so a healthy endpoint can complete without an installer nearby.

Supported auto-discovery filenames:

- `grabberEM.x64.msi`
- `grabberEM.x32.msi`
- `grabberTT.x64.msi`
- `grabberTT.x32.msi`
- `grabber.msi`

Lookup order:

1. Explicit `--installer`, when provided. Invalid explicit paths fail and do not fall back.
2. Directory next to `kigrepair.exe`.
3. Current working directory.
4. `.\assets\`
5. `<kigrepair.exe directory>\assets\`

On 64-bit Windows, x64 installers are preferred before x32 installers. On 32-bit Windows, x32 installers are preferred before x64 installers. EM is preferred over TT for the same architecture, and `grabber.msi` is the legacy fallback.

## JSON And Quiet Behavior

For commands that support `--json`, stdout contains exactly one JSON object. Human-readable output is suppressed and logs are written to `repair.log`.

`--quiet` suppresses human-readable console output, but does not suppress reports or logs. When `--json` and `--quiet` are used together, JSON is still printed to stdout.

## Non-Interactive Behavior

`--non-interactive` disables prompts. Destructive or system-changing workflows require `--yes` when non-interactive:

- `cleanup` requires `--yes`
- `repair` requires `--yes`
- `defender --ensure` requires `--yes`

`install` does not require `--yes` when a valid invite and installer are provided. All system-changing workflows still require administrator rights.

## Administrator Elevation

Admin-required commands can relaunch themselves through the standard Windows UAC prompt when started from a non-elevated interactive PowerShell:

- `cleanup` real mode
- `install`
- `defender --ensure`
- `repair`

Read-only commands do not request UAC: `check`, `cleanup --dry-run`, `defender` without `--ensure`, `collect-report`, and `version`.

Use `--no-elevate` to disable the UAC relaunch and return exit code `2` instead:

```powershell
.\kigrepair.exe repair --invite <INVITE> --installer .\grabber.msi --no-elevate
```

Quiet or non-interactive automation does not auto-prompt UAC. Run those commands from an elevated PowerShell, or omit `--non-interactive` when a support engineer is present to approve the UAC prompt.

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

`check` preserves health-oriented exit codes:

- `0` - healthy
- `1` - not installed
- `2` - broken
- `3` - partially removed
- `4` - unknown

## Safety Notes

- `check`, `cleanup --dry-run`, `defender` without `--ensure`, and `collect-report` are detection/reporting workflows and must not modify the system.
- Real cleanup, repair, install, and Defender ensure require administrator rights for system changes.
- Cleanup targets are centrally configured and validated before deletion.
- Registry cleanup uses allowlisted keys.
- Invite values must not be printed, logged, or written to JSON reports. Reports may contain `invite_provided: true/false` and masked command arguments such as `invite=***`.
- Quiet and non-interactive modes are intended for authorized support or enterprise automation and remain auditable through logs and reports.

## MVP Limitations

- Some Windows operations currently use `sc.exe`, `taskkill.exe`, PowerShell, `reg.exe`, and `msiexec.exe`.
- Native Windows APIs may replace service/process/registry operations later.
- Defender integration currently uses PowerShell cmdlets.
- GUI is not implemented.
- The tool is Windows-focused; tests avoid requiring real Defender, real services, or real MSI execution.

## Development

Requirements:

- Windows
- Go 1.22 or newer
- VS Code with the Go extension

Validation:

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
```

## Building a release

Run the release build from PowerShell:

```powershell
.\scripts\build.ps1 -Version 0.1.0
```

The script runs `gofmt`, `go mod tidy`, `go test ./...`, and a Windows amd64 build with embedded version metadata. Use `-SkipTests` only when you intentionally want a faster local package build:

```powershell
.\scripts\build.ps1 -Version 0.1.0 -SkipTests
```

Release output is written to:

```text
dist\kigrepair-v0.1.0-windows-amd64\
```

The package folder contains `kigrepair.exe`, `README.md`, `RELEASE_NOTES.md`, `checksums.txt`, and `kigrepair-v0.1.0-windows-amd64.zip`. If supported Grabber MSI files exist in the project root or `.\assets\`, the script copies all matching installers into the release folder. If no supported MSI is found, the build continues and packages the tool only.

`checksums.txt` contains SHA256 hashes for packaged files. The ZIP is created with PowerShell `Compress-Archive`; no external archive tool is required.

## Code signing

Code signing is not currently automated.
Future release process may sign `kigrepair.exe` before checksums and ZIP creation.

Manual non-destructive smoke checks:

```powershell
.\kigrepair.exe check
.\kigrepair.exe check --json
.\kigrepair.exe cleanup --dry-run
.\kigrepair.exe defender
.\kigrepair.exe collect-report
.\kigrepair.exe version
```

To write reports to a local directory during development:

```powershell
.\kigrepair.exe check --output .\Reports\dev-check
```

## Architecture

The project is split into internal packages for app context, workflows, configuration, detection, cleanup, installation, Defender integration, diagnostics, reports, safety validation, Windows-specific wrappers, logging, and future UI surfaces.

Keep workflow orchestration separate from low-level Windows operations. Destructive actions should go through planning, safety validation, operation result recording, logging, before/after detection, and report writing.
