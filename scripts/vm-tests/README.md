# kigrepair VM Test Scripts

These scripts support real-Windows VM validation. Run them from the release/test root that contains `kigrepair.exe`.

Read-only scripts:

- `run-smoke-readonly.ps1`
- `run-preflight-dryrun.ps1`
- `run-redaction-check.ps1`
- `collect-vm-evidence.ps1`
- `summarize-vm-results.ps1`

Guarded destructive template:

- `run-destructive-repair-template.ps1`

Scenario setup templates:

- `scenario-setup\*.ps1`

Do not store invite values in scripts, evidence, or docs. Use `$env:KIGREPAIR_TEST_INVITE` for local test sessions.

Validate script syntax:

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

