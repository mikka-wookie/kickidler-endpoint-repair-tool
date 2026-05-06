# kigrepair

`kigrepair` is a Windows-focused Kickidler support utility for Grabber diagnostics, cleanup, install, repair, Defender exclusion validation, verification, and support bundle collection.

The tool is CLI-first and intended for technical support engineers. Workflow reports are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

## Safety

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

## Common Commands

```powershell
.\kigrepair.exe version
.\kigrepair.exe config sample
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
.\kigrepair.exe collect-report
```

The support bundle is written as:

```text
kigrepair-support-bundle.zip
```

## Support Documentation

- [Support KB](docs/SUPPORT-KB.md)
- [Command Reference](docs/COMMAND-REFERENCE.md)
- [Configuration](docs/CONFIGURATION.md)
- [Report Files](docs/REPORT-FILES.md)
- [Classifications](docs/CLASSIFICATIONS.md)
- [Exit Codes](docs/EXIT-CODES.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Escalation Checklist](docs/ESCALATION-CHECKLIST.md)
- [Safety Model](docs/SAFETY-MODEL.md)
- [GUI Boundary](docs/GUI-BOUNDARY.md)
- [GUI MVP](docs/GUI-MVP.md)
- [Release Checklist](docs/RELEASE-CHECKLIST.md)
- [Release Trust](docs/RELEASE-TRUST.md)
- [VM Testing](docs/vm-testing/README.md)
- [Rollout](docs/ROLLOUT.md)
- [Known Limitations](docs/KNOWN-LIMITATIONS.md)
- [Short Support Runbook](SUPPORT-RUNBOOK.md)

## MSI Assets

Release packages include an empty `assets\` folder with instructions. Place a supported Grabber MSI there when using installer auto-discovery:

- `grabberEM.x64.msi`
- `grabberEM.x32.msi`
- `grabberTT.x64.msi`
- `grabberTT.x32.msi`
- `grabber.msi`

Do not place invite values in files.

## Release Build

From the repository root:

```powershell
.\scripts\build-release.ps1 -Version 0.1.0
```

The script runs formatting, dependency tidy, tests, and a Windows amd64 build with linker-injected metadata. It creates:

```text
dist\kigrepair-0.1.0-windows-amd64\
dist\kigrepair-0.1.0-windows-amd64.zip
dist\kigrepair-0.1.0-windows-amd64.zip.sha256
```

Release folder layout includes `kigrepair.exe`, optional `kigrepair-gui.exe`, `README.md`, `SUPPORT-RUNBOOK.md`, `docs\`, `assets\README.txt`, `examples\commands.ps1`, `checksums.txt`, `RELEASE-MANIFEST.json`, and `SIGNATURES.txt`.

Unsigned internal builds are allowed and clearly marked in `SIGNATURES.txt`. For signed support releases:

```powershell
.\scripts\build-release.ps1 -Version 0.1.0 -IncludeGui -Sign -CertificateThumbprint "<THUMBPRINT>" -TimestampUrl "<TIMESTAMP_URL>" -Clean
```

Validate a release folder and zip:

```powershell
.\scripts\validate-release.ps1 -ReleaseDir ".\dist\kigrepair-0.1.0-windows-amd64" -ZipPath ".\dist\kigrepair-0.1.0-windows-amd64.zip"
```

The release script does not bundle Grabber MSI installers. Support engineers must place approved MSI files locally when needed.

Before pilot or internal-stable promotion, run the real-Windows VM gates documented in [VM Testing](docs/vm-testing/README.md). From the repository validation workspace, Tier 0 smoke can be started against a built or unpacked release binary with:

```powershell
.\scripts\vm-tests\run-smoke-readonly.ps1 `
  -KigrepairPath ".\kigrepair.exe" `
  -OutDir ".\evidence\tier0-smoke" `
  -Profile standard
```

## Development Validation

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
go build -o kigrepair-gui.exe ./cmd/kigrepair-gui
```

## Architecture

The project is split into internal packages for app context, workflows, configuration, detection, cleanup, installation, Defender integration, diagnostics, reports, safety validation, Windows wrappers, logging, and future UI surfaces. Keep workflow orchestration separate from low-level Windows operations.

The GUI MVP uses the workflow service boundary in `internal/app` and `internal/app/workflowservice`. It must not shell out to `kigrepair.exe`, parse console output, or store invite values.
