param(
    [string]$Version = "",
    [string]$Commit = "",
    [string]$BuildDate = "",
    [string]$BuiltBy = "",
    [switch]$IncludeGui,
    [switch]$Sign,
    [string]$CertificateThumbprint = "",
    [string]$TimestampUrl = "http://timestamp.digicert.com",
    [string]$OutDir = "dist",
    [switch]$Clean,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

function Invoke-Step {
    param([string]$Name, [scriptblock]$Script)
    Write-Host ""
    Write-Host "== $Name =="
    & $Script
}

function Get-GitTagVersion {
    try {
        $tag = (& git describe --tags --exact-match 2>$null).Trim()
        if (-not [string]::IsNullOrWhiteSpace($tag)) { return $tag }
    } catch {}
    return "dev"
}

function Get-GitCommit {
    try {
        $value = (& git rev-parse --short HEAD 2>$null).Trim()
        if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
    } catch {}
    return "unknown"
}

function Get-DefaultBuiltBy {
    $user = $env:USERNAME
    if ([string]::IsNullOrWhiteSpace($user)) { $user = $env:USER }
    if ([string]::IsNullOrWhiteSpace($user)) { $user = "release-script" }
    if (-not [string]::IsNullOrWhiteSpace($env:COMPUTERNAME)) {
        return "$user@$env:COMPUTERNAME"
    }
    return $user
}

function Get-GoVersion {
    try {
        $value = (& go version).Trim()
        if (-not [string]::IsNullOrWhiteSpace($value)) { return $value.Split(" ")[2] }
    } catch {}
    return "unknown"
}

function Get-Signtool {
    $cmd = Get-Command signtool.exe -ErrorAction SilentlyContinue
    if ($null -ne $cmd) { return $cmd.Source }
    $kits = @(
        "${env:ProgramFiles(x86)}\Windows Kits\10\bin",
        "${env:ProgramFiles}\Windows Kits\10\bin"
    )
    foreach ($root in $kits) {
        if ([string]::IsNullOrWhiteSpace($root) -or -not (Test-Path -LiteralPath $root)) { continue }
        $match = Get-ChildItem -LiteralPath $root -Recurse -Filter signtool.exe -ErrorAction SilentlyContinue |
            Where-Object { $_.FullName -match "\\x64\\signtool\.exe$" } |
            Sort-Object FullName -Descending |
            Select-Object -First 1
        if ($null -ne $match) { return $match.FullName }
    }
    return ""
}

function Get-RelativePath {
    param([string]$BasePath, [string]$Path)
    $base = [System.IO.Path]::GetFullPath($BasePath)
    if (-not $base.EndsWith([System.IO.Path]::DirectorySeparatorChar)) {
        $base += [System.IO.Path]::DirectorySeparatorChar
    }
    $target = [System.IO.Path]::GetFullPath($Path)
    $baseUri = [System.Uri]::new($base)
    $targetUri = [System.Uri]::new($target)
    return [System.Uri]::UnescapeDataString($baseUri.MakeRelativeUri($targetUri).ToString()).Replace("\", "/")
}

function Get-FileRecord {
    param(
        [string]$ReleaseDir,
        [string]$Path,
        [string]$Type,
        [bool]$Signed,
        [bool]$SignatureVerified
    )
    $item = Get-Item -LiteralPath $Path
    $hash = Get-FileHash -LiteralPath $Path -Algorithm SHA256
    [ordered]@{
        path = Get-RelativePath -BasePath $ReleaseDir -Path $Path
        type = $Type
        sha256 = $hash.Hash.ToLowerInvariant()
        size_bytes = $item.Length
        signed = $Signed
        signature_verified = $SignatureVerified
    }
}

function Write-Checksums {
    param([string]$ReleaseDir)
    $checksumPath = Join-Path $ReleaseDir "checksums.txt"
    if (Test-Path -LiteralPath $checksumPath) { Remove-Item -LiteralPath $checksumPath -Force }
    $files = Get-ChildItem -LiteralPath $ReleaseDir -Recurse -File |
        Where-Object { $_.Name -ne "checksums.txt" } |
        Sort-Object FullName
    foreach ($file in $files) {
        $hash = Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256
        $rel = Get-RelativePath -BasePath $ReleaseDir -Path $file.FullName
        Add-Content -LiteralPath $checksumPath -Encoding ASCII -Value "$($hash.Hash.ToLowerInvariant())  $rel"
    }
}

function Write-Manifest {
    param(
        [string]$Path,
        [object[]]$Artifacts,
        [string[]]$IncludedDocs,
        [object]$Zip,
        [bool]$SigningRequested,
        [bool]$SigningSucceeded
    )
    $manifest = [ordered]@{
        product = "kigrepair"
        version = $script:Version
        commit = $script:Commit
        build_date = $script:BuildDate
        built_by = $script:BuiltBy
        go_version = $script:GoVersion
        target = [ordered]@{ os = "windows"; arch = "amd64" }
        checksum_algorithm = "SHA-256"
        signing = [ordered]@{
            requested = $SigningRequested
            succeeded = $SigningSucceeded
            timestamp_url = $(if ($SigningRequested) { $script:TimestampUrl } else { "" })
        }
        artifacts = $Artifacts
        zip = $Zip
        included_docs = $IncludedDocs
        security_notes = [ordered]@{
            msi_bundled = $false
            invite_included = $false
            reports_included = $false
        }
    }
    $json = $manifest | ConvertTo-Json -Depth 8
    Set-Content -LiteralPath $Path -Encoding UTF8 -Value $json
}

function Invoke-Signature {
    param([string]$Signtool, [string]$FilePath, [string]$SignaturesPath)
    $name = Split-Path -Leaf $FilePath
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "File: $name"
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Signed: requested"
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Thumbprint: $CertificateThumbprint"
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Timestamp URL: $TimestampUrl"

    $signArgs = @("sign", "/fd", "SHA256", "/tr", $TimestampUrl, "/td", "SHA256", "/sha1", $CertificateThumbprint, $FilePath)
    $signOutput = & $Signtool @signArgs 2>&1
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Sign command: signtool sign /fd SHA256 /tr <TIMESTAMP_URL> /td SHA256 /sha1 <THUMBPRINT> $name"
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value ($signOutput | Out-String)
    if ($LASTEXITCODE -ne 0) {
        Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Verification status: not run; signing failed"
        throw "Authenticode signing failed for $name"
    }

    $verifyArgs = @("verify", "/pa", "/v", $FilePath)
    $verifyOutput = & $Signtool @verifyArgs 2>&1
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Verify command: signtool verify /pa /v $name"
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value ($verifyOutput | Out-String)
    if ($LASTEXITCODE -ne 0) {
        Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Verification status: failed"
        throw "Authenticode verification failed for $name"
    }
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value "Verification status: passed"
    Add-Content -LiteralPath $SignaturesPath -Encoding UTF8 -Value ""
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if ([string]::IsNullOrWhiteSpace($Version)) { $Version = Get-GitTagVersion }
if ([string]::IsNullOrWhiteSpace($Commit)) { $Commit = Get-GitCommit }
if ([string]::IsNullOrWhiteSpace($BuildDate)) { $BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ") }
if ([string]::IsNullOrWhiteSpace($BuiltBy)) { $BuiltBy = Get-DefaultBuiltBy }
if ([string]::IsNullOrWhiteSpace($Version)) { throw "Version cannot be empty." }

$GoVersion = Get-GoVersion
$packageName = "kigrepair-$Version-windows-amd64"
$resolvedOutDir = Join-Path $repoRoot $OutDir
$releaseDir = Join-Path $resolvedOutDir $packageName
$zipPath = Join-Path $resolvedOutDir "$packageName.zip"
$zipChecksumPath = "$zipPath.sha256"
$cliPath = Join-Path $releaseDir "kigrepair.exe"
$guiPath = Join-Path $releaseDir "kigrepair-gui.exe"
$manifestPath = Join-Path $releaseDir "RELEASE-MANIFEST.json"
$signaturesPath = Join-Path $releaseDir "SIGNATURES.txt"
$signatureByFile = @{}

if ($Sign -and [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
    throw "-Sign requires -CertificateThumbprint. Refusing to produce an unsigned release when signing was requested."
}

Invoke-Step "Preparing release directory" {
    if ($Clean -and (Test-Path -LiteralPath $resolvedOutDir)) {
        Remove-Item -LiteralPath $resolvedOutDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
    Get-ChildItem -LiteralPath $releaseDir -Force -ErrorAction SilentlyContinue | Remove-Item -Recurse -Force
    if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
    if (Test-Path -LiteralPath $zipChecksumPath) { Remove-Item -LiteralPath $zipChecksumPath -Force }
}

Invoke-Step "Running Go validation" {
    gofmt -w .
    go mod tidy
    if ($SkipTests) {
        Write-Warning "Skipping go test ./..."
    } else {
        go test ./...
    }
}

$ldflags = "-X kigrepair/internal/version.Version=$Version -X kigrepair/internal/version.Commit=$Commit -X kigrepair/internal/version.BuildDate=$BuildDate -X kigrepair/internal/version.BuiltBy=$BuiltBy"

Invoke-Step "Building CLI" {
    $oldGOOS = $env:GOOS
    $oldGOARCH = $env:GOARCH
    try {
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        go build -trimpath -ldflags $ldflags -o $cliPath ./cmd/kigrepair
    } finally {
        $env:GOOS = $oldGOOS
        $env:GOARCH = $oldGOARCH
    }
}

if ($IncludeGui) {
    Invoke-Step "Building GUI" {
        $oldGOOS = $env:GOOS
        $oldGOARCH = $env:GOARCH
        try {
            $env:GOOS = "windows"
            $env:GOARCH = "amd64"
            go build -trimpath -ldflags $ldflags -o $guiPath ./cmd/kigrepair-gui
        } finally {
            $env:GOOS = $oldGOOS
            $env:GOARCH = $oldGOARCH
        }
    }
} else {
    Write-Warning "GUI binary not included. Use -IncludeGui when a GUI release is required."
}

Invoke-Step "Copying release files" {
    Copy-Item -LiteralPath (Join-Path $repoRoot "README.md") -Destination (Join-Path $releaseDir "README.md") -Force
    Copy-Item -LiteralPath (Join-Path $repoRoot "SUPPORT-RUNBOOK.md") -Destination (Join-Path $releaseDir "SUPPORT-RUNBOOK.md") -Force
    Copy-Item -LiteralPath (Join-Path $repoRoot "docs") -Destination (Join-Path $releaseDir "docs") -Recurse -Force
    New-Item -ItemType Directory -Path (Join-Path $releaseDir "assets") -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot "assets\README.txt") -Destination (Join-Path $releaseDir "assets\README.txt") -Force
    New-Item -ItemType Directory -Path (Join-Path $releaseDir "examples") -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot "examples\commands.ps1") -Destination (Join-Path $releaseDir "examples\commands.ps1") -Force
}

if ($Sign) {
    Invoke-Step "Signing executables" {
        $signtool = Get-Signtool
        if ([string]::IsNullOrWhiteSpace($signtool)) {
            Set-Content -LiteralPath $signaturesPath -Encoding UTF8 -Value "Signing requested, but signtool.exe was not found."
            throw "Signing requested, but signtool.exe was not found."
        }
        Set-Content -LiteralPath $signaturesPath -Encoding UTF8 -Value "Authenticode signature results"
        foreach ($exe in @($cliPath, $guiPath)) {
            if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { continue }
            Invoke-Signature -Signtool $signtool -FilePath $exe -SignaturesPath $signaturesPath
            $signatureByFile[(Split-Path -Leaf $exe)] = @{ signed = $true; verified = $true }
        }
    }
} else {
    Invoke-Step "Recording unsigned release warning" {
        Set-Content -LiteralPath $signaturesPath -Encoding UTF8 -Value @(
            "Authenticode signature results",
            "Signing requested: false",
            "Warning: executables in this release were not signed by this script.",
            "Use -Sign -CertificateThumbprint <THUMBPRINT> for a signed support release."
        )
    }
}

Invoke-Step "Generating manifest and checksums" {
    $artifacts = @()
    $cliSig = $signatureByFile["kigrepair.exe"]
    $artifacts += Get-FileRecord -ReleaseDir $releaseDir -Path $cliPath -Type "executable" -Signed ([bool]$cliSig.signed) -SignatureVerified ([bool]$cliSig.verified)
    if (Test-Path -LiteralPath $guiPath -PathType Leaf) {
        $guiSig = $signatureByFile["kigrepair-gui.exe"]
        $artifacts += Get-FileRecord -ReleaseDir $releaseDir -Path $guiPath -Type "executable" -Signed ([bool]$guiSig.signed) -SignatureVerified ([bool]$guiSig.verified)
    }
    $includedDocs = Get-ChildItem -LiteralPath $releaseDir -Recurse -File |
        Where-Object { $_.Extension -in @(".md", ".txt", ".ps1") -and $_.FullName -notlike "*\SIGNATURES.txt" } |
        ForEach-Object { Get-RelativePath -BasePath $releaseDir -Path $_.FullName } |
        Sort-Object
    Write-Manifest -Path $manifestPath -Artifacts $artifacts -IncludedDocs $includedDocs -Zip ([ordered]@{ path = "../$packageName.zip"; sha256 = ""; size_bytes = 0 }) -SigningRequested ([bool]$Sign) -SigningSucceeded ([bool]$Sign)
    Write-Checksums -ReleaseDir $releaseDir
}

Invoke-Step "Creating ZIP archive" {
    Compress-Archive -Path (Join-Path $releaseDir "*") -DestinationPath $zipPath -Force
    $zipHash = Get-FileHash -LiteralPath $zipPath -Algorithm SHA256
    Set-Content -LiteralPath $zipChecksumPath -Encoding ASCII -Value "$($zipHash.Hash.ToLowerInvariant())  $packageName.zip"
}

Invoke-Step "Validating release output" {
    & (Join-Path $PSScriptRoot "validate-release.ps1") -ReleaseDir $releaseDir -ZipPath $zipPath
    & $cliPath version --json | Out-Null
}

Write-Host ""
Write-Host "Release package created"
Write-Host "Folder: $releaseDir"
Write-Host "Zip:    $zipPath"
Write-Host "SHA256: $zipChecksumPath"
if ($Sign) {
    Write-Host "Signing: requested and verified"
} else {
    Write-Warning "Signing: not requested. This is an unsigned release artifact."
}
