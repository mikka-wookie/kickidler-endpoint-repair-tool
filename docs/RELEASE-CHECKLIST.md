# kigrepair Release Checklist

Use this checklist before distributing a new internal build.

## Command Safety Classes

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Build Validation

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
```

## Build Release

```powershell
.\scripts\build-release.ps1 -Version 0.1.0
```

## Verify Version Metadata

```powershell
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe version
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe version --json
```

Confirm version, commit, build date, built by, Go version, OS, and architecture.

## Inspect Checksums

```powershell
Get-Content .\dist\kigrepair-0.1.0-windows-amd64\checksums.txt
Get-Content .\dist\kigrepair-0.1.0-windows-amd64\checksums.json
```

Confirm `kigrepair.exe` and the release ZIP are present.

## Inspect Release Archive

```powershell
Expand-Archive .\dist\kigrepair-0.1.0-windows-amd64.zip -DestinationPath "$env:TEMP\kigrepair-release-test" -Force
Get-ChildItem "$env:TEMP\kigrepair-release-test" -Recurse
```

Expected layout:

```text
dist/
  kigrepair-0.1.0-windows-amd64/
    kigrepair.exe
    README.md
    SUPPORT-RUNBOOK.md
    docs/
      SUPPORT-KB.md
      COMMAND-REFERENCE.md
      REPORT-FILES.md
      CLASSIFICATIONS.md
      EXIT-CODES.md
      TROUBLESHOOTING.md
      ESCALATION-CHECKLIST.md
      SAFETY-MODEL.md
      RELEASE-CHECKLIST.md
    assets/
      README.txt
    examples/
      commands.ps1
    checksums.txt
    checksums.json
```

## Read-only Smoke Tests

```powershell
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe version
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe check --help
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe repair --help
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe reports --help
```

## Documentation Checks

- Verify docs are included in the release folder and ZIP.
- Verify README links to every file in `docs\`.
- Verify `SUPPORT-RUNBOOK.md` is short and operational.
- Verify no real invite values or secrets are in the archive.
- Verify all examples use `<INVITE>`.
- Verify `assets\README.txt` is present.
- Verify `examples\commands.ps1` is present.

## Final Review

- Run tests.
- Build release.
- Verify version metadata.
- Check checksums.
- Inspect release archive.
- Run read-only smoke tests.
- Verify docs included.
- Verify no real invite/secrets in archive.
- Verify `assets\README.txt`.
- Verify release ZIP extraction.
