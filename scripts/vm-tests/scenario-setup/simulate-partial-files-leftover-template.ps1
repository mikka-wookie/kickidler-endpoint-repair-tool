param(
    [ValidateSet("standard", "helper", "programfiles_x86", "helper_x86", "programdata_exact", "hidden_wmi")]
    [string]$Target = "standard",
    [switch]$DryRun,
    [switch]$IUnderstandThisIsDestructive
)

$ErrorActionPreference = "Stop"

$targets = @{
    standard = "$env:ProgramFiles\TeleLinkSoft"
    helper = "$env:ProgramFiles\TeleLinkSoftHelper"
    programfiles_x86 = "${env:ProgramFiles(x86)}\TeleLinkSoft"
    helper_x86 = "${env:ProgramFiles(x86)}\TeleLinkSoftHelper"
    programdata_exact = "$env:ProgramData\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60"
    hidden_wmi = "$env:SystemRoot\System32\wmi"
}

$path = $targets[$Target]
Write-Warning "Template only: simulate partial files leftover on a disposable VM snapshot."
Write-Host "Allowlisted target: $path"
Write-Host "Planned setup: create a harmless marker file only under the exact selected allowlisted path after approval."
Write-Host "This template does not delete files and does not touch arbitrary Windows paths."

if (-not $DryRun) {
    if (-not $IUnderstandThisIsDestructive) {
        Write-Error "Refusing mutation. Re-run with -DryRun for planning or pass -IUnderstandThisIsDestructive after manual approval."
        exit 1
    }
    Write-Error "Mutation is intentionally not implemented in this template. Create the approved leftover state manually if needed."
    exit 1
}

Write-Host "Dry-run complete. No changes made."

