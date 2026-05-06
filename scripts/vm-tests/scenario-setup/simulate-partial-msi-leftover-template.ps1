param(
    [switch]$DryRun,
    [switch]$IUnderstandThisIsDestructive
)

$ErrorActionPreference = "Stop"

$keys = @(
    "HKCR\Installer\Features\73CBF1BE79B05FC43A92FC82AB567384",
    "HKCR\Installer\Products\73CBF1BE79B05FC43A92FC82AB567384",
    "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}",
    "HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}",
    "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products\73CBF1BE79B05FC43A92FC82AB567384"
)

Write-Warning "Template only: simulate partial MSI leftovers on a disposable VM snapshot."
Write-Host "Known MSI keys that may be inspected in this scenario:"
$keys | ForEach-Object { Write-Host "  $_" }
Write-Host "This template does not create or delete registry keys."

if (-not $DryRun) {
    if (-not $IUnderstandThisIsDestructive) {
        Write-Error "Refusing mutation. Re-run with -DryRun for planning or pass -IUnderstandThisIsDestructive after manual approval."
        exit 1
    }
    Write-Error "Mutation is intentionally not implemented in this template. Create the approved MSI leftover state manually if needed."
    exit 1
}

Write-Host "Dry-run complete. No changes made."

