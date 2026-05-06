# kigrepair Support Runbook

Use this as the short operational entry point. Detailed KB content is in [docs/SUPPORT-KB.md](docs/SUPPORT-KB.md), and command details are in [docs/COMMAND-REFERENCE.md](docs/COMMAND-REFERENCE.md).

Reports are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Optional team defaults can be placed in `kigrepair.yaml`:

```powershell
.\kigrepair.exe config sample > .\kigrepair.yaml
.\kigrepair.exe config validate --config ".\kigrepair.yaml"
```

Never store invite values or secrets in configuration.

Select a policy profile per case:

```powershell
.\kigrepair.exe check --profile conservative
.\kigrepair.exe verify --profile diagnostic
.\kigrepair.exe repair --dry-run --profile conservative --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe collect-report --profile diagnostic
```

`standard` is normal support usage, `conservative` is for uncertain or high-risk endpoints, and `diagnostic` favors read-only evidence collection. CLI `--profile` overrides YAML for that run.

## Safety

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

Failure-mode rule: if a command fails before making changes, review `summary.txt`, `operations.json`, and the first failed operation. Expected blockers should return `7`; warning or partial diagnostics should return `1`; unexpected tool/report infrastructure failures should return `10`.

## Observability

Use the `Run ID` in `summary.txt` to correlate `operations.json`, `repair.log`, and result files. `repair.log` is JSON Lines and is safe to parse or attach after confirming redaction. The `Workflow timeline` section in `summary.txt` is the quickest support-readable reconstruction of what happened.

Reports and bundles should redact raw invite values and common secrets. If a raw invite appears, do not attach the file until it is sanitized.

The GUI MVP is available as `kigrepair-gui.exe` for support engineers who prefer a single-window workflow shell. It uses the shared workflow service and the same report files; it must not parse CLI console output or store invite values.

```powershell
.\kigrepair-gui.exe
```

For release verification, check `RELEASE-MANIFEST.json`, `checksums.txt`, and `SIGNATURES.txt`, or run:

```powershell
.\scripts\validate-release.ps1 -ReleaseDir ".\dist\kigrepair-<version>-windows-amd64"
```

## 1. Basic Triage

```powershell
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

Review `summary.txt`, `classification-result.json`, and `verification-result.json`.

## 2. Safe Preview

```powershell
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Review `preflight-result.json`, `repair-plan.json`, and `cleanup-plan.json`.

Confirm the active profile and policy decisions in `config-metadata.json`, `preflight-result.json`, and `repair-plan.json`.

## 3. Real Repair

Destructive: run only when approved and from an elevated PowerShell.

```powershell
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

## 4. After Repair

```powershell
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

Attach `kigrepair-support-bundle.zip` when escalating.

## 5. Report Cleanup

```powershell
.\kigrepair.exe reports cleanup --dry-run
.\kigrepair.exe reports cleanup --yes
```

Use dry-run first. Real report cleanup deletes only validated old report folders under the report root.

## Detailed References

- [Report Files](docs/REPORT-FILES.md)
- [Classifications](docs/CLASSIFICATIONS.md)
- [Exit Codes](docs/EXIT-CODES.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Escalation Checklist](docs/ESCALATION-CHECKLIST.md)
- [Safety Model](docs/SAFETY-MODEL.md)
- [GUI Boundary](docs/GUI-BOUNDARY.md)
