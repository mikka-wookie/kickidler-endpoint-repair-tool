param(
    [string]$ReportsRoot = "C:\ProgramData\kigrepair\Reports",
    [switch]$DryRun,
    [switch]$IUnderstandThisIsDestructive
)

$ErrorActionPreference = "Stop"

$expectedRoot = "C:\ProgramData\kigrepair\Reports"
if ([System.IO.Path]::GetFullPath($ReportsRoot).TrimEnd("\") -ine $expectedRoot) {
    Write-Error "Refusing unknown report root. Expected exact path: $expectedRoot"
    exit 1
}

Write-Warning "Template only: simulate stale report retention under the exact kigrepair report root."
Write-Host "Allowlisted report root: $expectedRoot"
Write-Host "Planned setup: create old report-like folders under the report root, then run reports cleanup --dry-run."
Write-Host "This template does not delete report folders."

if (-not $DryRun) {
    if (-not $IUnderstandThisIsDestructive) {
        Write-Error "Refusing mutation. Re-run with -DryRun for planning or pass -IUnderstandThisIsDestructive after manual approval."
        exit 1
    }
    Write-Error "Mutation is intentionally not implemented in this template. Create approved stale report folders manually if needed."
    exit 1
}

Write-Host "Dry-run complete. No changes made."

