# VM Smoke Tests

## CLI Read-Only Smoke

Run from the VM test directory:

```powershell
.\scripts\vm-tests\run-smoke-readonly.ps1 `
  -KigrepairPath ".\kigrepair.exe" `
  -OutDir ".\evidence\tier0-smoke" `
  -Profile standard
```

The script runs only read-only commands except normal report/evidence writes:

- `version --json`
- `config show`
- `check`
- `verify`
- `cleanup --dry-run`
- `collect-report`
- `reports list`

Expected:

- `vm-smoke-result.json` is written.
- `command-output\` contains stdout/stderr per command.
- No invite is required.
- No installer is required.
- No service, process, Defender, MSI, registry, product file, or report cleanup mutation occurs.

## Preflight and Repair Dry-Run

Use an approved installer and local env-var invite:

```powershell
$env:KIGREPAIR_TEST_INVITE = "<INVITE>"

.\scripts\vm-tests\run-preflight-dryrun.ps1 `
  -KigrepairPath ".\kigrepair.exe" `
  -InstallerPath ".\assets\grabberEM.x64.msi" `
  -OutDir ".\evidence\preflight-dryrun" `
  -Profile conservative
```

Expected:

- The script fails safely if the env var is missing.
- The script does not print or write the invite.
- Captured command output is redacted before it is written.
- No mutation occurs.

## Redaction Check

Use a fake invite value in a disposable VM:

```powershell
Set-Item Env:KIGREPAIR_TEST_INVITE "REAL-SECRET-INVITE"

.\scripts\vm-tests\run-preflight-dryrun.ps1 `
  -KigrepairPath ".\kigrepair.exe" `
  -InstallerPath ".\assets\grabberEM.x64.msi" `
  -OutDir ".\evidence\redaction" `
  -Profile conservative

.\kigrepair.exe collect-report

.\scripts\vm-tests\run-redaction-check.ps1 `
  -Path ".\evidence" `
  -SecretPattern "REAL-SECRET-INVITE"

.\scripts\vm-tests\run-redaction-check.ps1 `
  -Path "C:\ProgramData\kigrepair\Reports" `
  -SecretPattern "REAL-SECRET-INVITE"
```

Expected:

- No raw fake invite in reports.
- No raw fake invite in support bundle.
- No raw fake invite in evidence output.
- Command output uses `<REDACTED>` if the invite appears conceptually.

## GUI MVP Smoke

Required if `kigrepair-gui.exe` is distributed:

- Launch GUI.
- Confirm version/profile are shown.
- Run Check.
- Run Verify.
- Run Collect Report.
- Run Preflight with masked invite.
- Run Repair Dry-Run.
- Confirm timeline updates.
- Confirm report directory link works.
- Cancel a running workflow if practical.
- Confirm invite field is masked.
- Confirm invite field is cleared after workflow completion or cancellation where practical.
- Confirm real repair requires exact `YES`.
- Confirm no raw invite appears in reports, evidence, screenshots, timeline, title, or status text.

No customer-facing UX claims should be made from MVP smoke results.

## PowerShell Syntax Validation

```powershell
Get-ChildItem .\scripts\vm-tests\*.ps1 -Recurse | ForEach-Object {
    $errors = $null
    $null = [System.Management.Automation.PSParser]::Tokenize((Get-Content $_.FullName -Raw), [ref]$errors)
    if ($errors) {
        Write-Error "PowerShell parse errors in $($_.FullName): $errors"
        exit 1
    }
}
```
