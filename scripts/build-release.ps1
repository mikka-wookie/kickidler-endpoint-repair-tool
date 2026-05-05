param(
    [string]$Version = "dev",
    [string]$OutputDir = "dist",
    [switch]$SkipTests,
    [ValidateSet("amd64")]
    [string]$Arch = "amd64"
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

function Get-BuiltBy {
    if (-not [string]::IsNullOrWhiteSpace($env:USERNAME)) {
        return $env:USERNAME
    }
    if (-not [string]::IsNullOrWhiteSpace($env:USER)) {
        return $env:USER
    }
    return "release-script"
}

function Write-TextFile {
    param(
        [string]$Path,
        [string]$Content,
        [System.Text.Encoding]$Encoding = [System.Text.Encoding]::UTF8
    )

    $parent = Split-Path -Parent $Path
    if (-not [string]::IsNullOrWhiteSpace($parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    [System.IO.File]::WriteAllText($Path, $Content, $Encoding)
}

function Add-ChecksumLine {
    param(
        [string]$ChecksumPath,
        [string]$FilePath,
        [string]$DisplayName
    )

    $hash = Get-FileHash -LiteralPath $FilePath -Algorithm SHA256
    Add-Content -LiteralPath $ChecksumPath -Encoding ASCII -Value "SHA256  $DisplayName  $($hash.Hash)"
    return @{
        algorithm = "SHA256"
        file = $DisplayName
        hash = $hash.Hash
    }
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if ([string]::IsNullOrWhiteSpace($Version)) {
    throw "Version cannot be empty"
}

$targetOS = "windows"
$packageName = "kigrepair-$Version-$targetOS-$Arch"
$resolvedOutputDir = Join-Path $repoRoot $OutputDir
$releaseDir = Join-Path $resolvedOutputDir $packageName
$binaryPath = Join-Path $releaseDir "kigrepair.exe"
$zipPath = Join-Path $resolvedOutputDir "$packageName.zip"
$checksumPath = Join-Path $releaseDir "checksums.txt"
$checksumsJsonPath = Join-Path $releaseDir "checksums.json"
$commit = Get-GitCommit
$buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$builtBy = Get-BuiltBy

Invoke-Step "Preparing release directory: $releaseDir" {
    New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
    Get-ChildItem -LiteralPath $releaseDir -Force | Remove-Item -Recurse -Force
    if (Test-Path -LiteralPath $zipPath -PathType Leaf) {
        Remove-Item -LiteralPath $zipPath -Force
    }
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

$ldflags = @(
    "-X", "kigrepair/internal/version.Version=$Version",
    "-X", "kigrepair/internal/version.Commit=$commit",
    "-X", "kigrepair/internal/version.BuildDate=$buildDate",
    "-X", "kigrepair/internal/version.BuiltBy=$builtBy"
) -join " "

Invoke-Step "Building $binaryPath" {
    $previousGOOS = $env:GOOS
    $previousGOARCH = $env:GOARCH
    try {
        $env:GOOS = $targetOS
        $env:GOARCH = $Arch
        go build -trimpath -ldflags $ldflags -o $binaryPath ./cmd/kigrepair
    } finally {
        $env:GOOS = $previousGOOS
        $env:GOARCH = $previousGOARCH
    }
}

Invoke-Step "Copying release documentation" {
    Copy-Item -LiteralPath (Join-Path $repoRoot "README.md") -Destination (Join-Path $releaseDir "README.md") -Force
    Copy-Item -LiteralPath (Join-Path $repoRoot "SUPPORT-RUNBOOK.md") -Destination (Join-Path $releaseDir "SUPPORT-RUNBOOK.md") -Force
    New-Item -ItemType Directory -Path (Join-Path $releaseDir "assets") -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot "assets\README.txt") -Destination (Join-Path $releaseDir "assets\README.txt") -Force
    New-Item -ItemType Directory -Path (Join-Path $releaseDir "examples") -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot "examples\commands.ps1") -Destination (Join-Path $releaseDir "examples\commands.ps1") -Force
}

Invoke-Step "Generating executable checksum" {
    if (Test-Path -LiteralPath $checksumPath -PathType Leaf) {
        Remove-Item -LiteralPath $checksumPath -Force
    }
    $checksums = @()
    $checksums += Add-ChecksumLine -ChecksumPath $checksumPath -FilePath $binaryPath -DisplayName "kigrepair.exe"
    $checksums | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $checksumsJsonPath -Encoding UTF8
}

Invoke-Step "Creating ZIP archive" {
    Compress-Archive -LiteralPath $releaseDir -DestinationPath $zipPath -Force
}

Invoke-Step "Adding ZIP checksum" {
    $checksums = Get-Content -LiteralPath $checksumsJsonPath -Raw | ConvertFrom-Json
    $checksums = @($checksums)
    $checksums += Add-ChecksumLine -ChecksumPath $checksumPath -FilePath $zipPath -DisplayName "$packageName.zip"
    $checksums | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $checksumsJsonPath -Encoding UTF8
}

Invoke-Step "Validating release binary metadata" {
    & $binaryPath version
    & $binaryPath version --json | Out-Null
    & $binaryPath check --help | Out-Null
    & $binaryPath repair --help | Out-Null
    & $binaryPath reports --help | Out-Null
}

Write-Host ""
Write-Host "Release package created:"
Write-Host $releaseDir
Write-Host $zipPath
