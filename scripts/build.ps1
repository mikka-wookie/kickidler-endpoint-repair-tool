param(
    [string]$Version = "dev",
    [string]$OutputDir = "dist",
    [switch]$SkipTests,
    [ValidateSet("amd64")]
    [string]$Arch = "amd64"
)

$ErrorActionPreference = "Stop"

& (Join-Path $PSScriptRoot "build-release.ps1") -Version $Version -OutputDir $OutputDir -SkipTests:$SkipTests -Arch $Arch
