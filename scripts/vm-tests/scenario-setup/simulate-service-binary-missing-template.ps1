param(
    [string]$ServiceName = "ngs",
    [switch]$DryRun,
    [switch]$IUnderstandThisIsDestructive
)

$ErrorActionPreference = "Stop"

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

Write-Warning "Template only: simulate a service binary missing scenario on a disposable VM snapshot."
Write-Host "Planned scenario: inspect service '$ServiceName', identify its ImagePath, then manually rename only that exact service executable after approval."
Write-Host "Normal Windows process binaries must never be modified."

if (-not $DryRun) {
    if (-not $IUnderstandThisIsDestructive) {
        Write-Error "Refusing mutation. Re-run with -DryRun for planning or pass -IUnderstandThisIsDestructive after manual approval."
        exit 1
    }
    if (-not (Test-IsAdmin)) {
        Write-Error "Refusing mutation because this shell is not elevated."
        exit 1
    }
    Write-Error "Mutation is intentionally not implemented in this template. Perform the exact approved setup manually on a disposable snapshot."
    exit 1
}

Write-Host "Dry-run complete. No changes made."

