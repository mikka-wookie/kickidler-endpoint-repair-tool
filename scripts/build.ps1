param(
    [string]$Version = "0.1.0",
    [string]$OutputDir = ".\dist",
    [bool]$IncludeMSI = $true,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Script
    )

    Write-Host $Name
    & $Script
}

function Get-GitCommit {
    try {
        $commit = (& git rev-parse --short HEAD 2>$null).Trim()
        if ([string]::IsNullOrWhiteSpace($commit)) {
            return "unknown"
        }
        return $commit
    } catch {
        return "unknown"
    }
}

function Copy-IfExists {
    param(
        [string]$Source,
        [string]$Destination
    )

    if (Test-Path -LiteralPath $Source -PathType Leaf) {
        Copy-Item -LiteralPath $Source -Destination $Destination -Force
        return $true
    }
    return $false
}

function Get-SupportedInstallerNames {
    return @(
        "grabberEM.x64.msi",
        "grabberEM.x32.msi",
        "grabberTT.x64.msi",
        "grabberTT.x32.msi",
        "grabber.msi"
    )
}

function Get-ReleaseNotesTemplate {
    param([string]$ReleaseVersion)

    return @"
# kigrepair v$ReleaseVersion

## Summary

Windows support utility for Kickidler Grabber diagnostics, cleanup, install, Defender exclusion handling, repair orchestration, and support bundle collection.

## Included commands

- check
- cleanup --dry-run
- cleanup
- install
- defender
- repair
- collect-report
- version

## Safety notes

- Detection and collect-report are read-only.
- cleanup --dry-run is read-only.
- cleanup, install, defender --ensure, and repair require administrator rights.
- Interactive mode can request UAC elevation.
- quiet/non-interactive mode does not trigger UAC and requires elevated shell.
- Raw invite values are not logged.

## MVP limitations

- Some service/process operations may use sc.exe, taskkill.exe, and PowerShell.
- Native Windows APIs may replace these later.
- GUI is not implemented yet.
- Code signing is not implemented by this script unless added later.

## Validation

- Build date:
- Commit:
- Tested on:
- Notes:
"@
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$targetOS = "windows"
$targetArch = "amd64"
$packageName = "kigrepair-v$Version-$targetOS-$targetArch"
$resolvedOutputDir = Join-Path $repoRoot $OutputDir
$releaseDir = Join-Path $resolvedOutputDir $packageName
$zipPath = Join-Path $releaseDir "$packageName.zip"
$binaryPath = Join-Path $releaseDir "kigrepair.exe"
$commit = Get-GitCommit
$buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

if ([string]::IsNullOrWhiteSpace($Version)) {
    throw "Version cannot be empty"
}

if ([string]::IsNullOrWhiteSpace($packageName)) {
    throw "Package name cannot be empty"
}

Invoke-Step "Preparing release directory: $releaseDir" {
    New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
    Get-ChildItem -LiteralPath $releaseDir -Force | Remove-Item -Recurse -Force
}

Invoke-Step "Running gofmt" {
    gofmt -w .
}

Invoke-Step "Running go mod tidy" {
    go mod tidy
}

if (-not $SkipTests) {
    Invoke-Step "Running go test ./..." {
        go test ./...
    }
} else {
    Write-Warning "Skipping go test ./..."
}

$ldflags = "-X kigrepair/internal/config.Version=$Version -X kigrepair/internal/config.GitCommit=$commit -X kigrepair/internal/config.BuildDate=$buildDate"

Invoke-Step "Building $binaryPath" {
    $env:GOOS = $targetOS
    $env:GOARCH = $targetArch
    go build -ldflags $ldflags -o $binaryPath ./cmd/kigrepair
}

Invoke-Step "Copying documentation" {
    Copy-Item -LiteralPath (Join-Path $repoRoot "README.md") -Destination (Join-Path $releaseDir "README.md") -Force
    $releaseNotesPath = Join-Path $repoRoot "RELEASE_NOTES.md"
    if (Test-Path -LiteralPath $releaseNotesPath -PathType Leaf) {
        $releaseNotes = Get-Content -LiteralPath $releaseNotesPath -Raw
        $releaseNotes = $releaseNotes -replace '(?m)^# kigrepair v\S+', "# kigrepair v$Version"
    } else {
        Write-Warning "RELEASE_NOTES.md not found; creating release notes from template"
        $releaseNotes = Get-ReleaseNotesTemplate -ReleaseVersion $Version
    }
    Set-Content -LiteralPath (Join-Path $releaseDir "RELEASE_NOTES.md") -Value $releaseNotes -Encoding UTF8
}

if ($IncludeMSI) {
    $copiedInstallers = @{}
    $installerSearchDirs = @(
        $repoRoot,
        (Join-Path $repoRoot "assets")
    )
    foreach ($dir in $installerSearchDirs) {
        foreach ($name in Get-SupportedInstallerNames) {
            $candidate = Join-Path $dir $name
            if ((-not $copiedInstallers.ContainsKey($name)) -and (Copy-IfExists -Source $candidate -Destination (Join-Path $releaseDir $name))) {
                $copiedInstallers[$name] = $true
            }
        }
    }
    if ($copiedInstallers.Count -eq 0) {
        Write-Warning "no Grabber MSI installers found; release package will contain kigrepair.exe only"
    }
}

# Code signing can be inserted here before checksums and ZIP creation when a certificate is configured.

Invoke-Step "Generating checksums" {
    $checksumPath = Join-Path $releaseDir "checksums.txt"
    $filesToHash = @("kigrepair.exe", "README.md", "RELEASE_NOTES.md") + (Get-SupportedInstallerNames)
    $lines = foreach ($name in $filesToHash) {
        $path = Join-Path $releaseDir $name
        if (Test-Path -LiteralPath $path -PathType Leaf) {
            $hash = Get-FileHash -LiteralPath $path -Algorithm SHA256
            "SHA256  $name  $($hash.Hash)"
        }
    }
    $lines | Set-Content -LiteralPath $checksumPath -Encoding ASCII
}

Invoke-Step "Creating ZIP package" {
    if (Test-Path -LiteralPath $zipPath -PathType Leaf) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    $zipItems = Get-ChildItem -LiteralPath $releaseDir -File | Where-Object { $_.FullName -ne $zipPath }
    Compress-Archive -LiteralPath $zipItems.FullName -DestinationPath $zipPath -Force
}

Invoke-Step "Adding ZIP checksum" {
    $checksumPath = Join-Path $releaseDir "checksums.txt"
    $hash = Get-FileHash -LiteralPath $zipPath -Algorithm SHA256
    Add-Content -LiteralPath $checksumPath -Encoding ASCII -Value "SHA256  $packageName.zip  $($hash.Hash)"
}

Write-Host ""
Write-Host "Release package created:"
Write-Host $releaseDir
