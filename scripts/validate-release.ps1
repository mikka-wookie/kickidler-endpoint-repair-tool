param(
    [string]$ReleaseDir,
    [string]$ZipPath = "",
    [switch]$RequireSigned
)

$ErrorActionPreference = "Stop"

function Fail {
    param([string]$Message)
    throw "Release validation failed: $Message"
}

function Get-RelativePath {
    param([string]$BasePath, [string]$Path)
    $base = [System.IO.Path]::GetFullPath($BasePath)
    if (-not $base.EndsWith([System.IO.Path]::DirectorySeparatorChar)) {
        $base += [System.IO.Path]::DirectorySeparatorChar
    }
    $baseUri = [System.Uri]::new($base)
    $targetUri = [System.Uri]::new([System.IO.Path]::GetFullPath($Path))
    return [System.Uri]::UnescapeDataString($baseUri.MakeRelativeUri($targetUri).ToString()).Replace("\", "/")
}

function Test-RequiredFile {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        Fail "missing required file: $Path"
    }
}

function Assert-ChecksumFile {
    param([string]$ReleaseDir)
    $checksumPath = Join-Path $ReleaseDir "checksums.txt"
    Test-RequiredFile $checksumPath
    $lines = Get-Content -LiteralPath $checksumPath | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    if ($lines.Count -eq 0) { Fail "checksums.txt is empty" }
    foreach ($line in $lines) {
        if ($line -notmatch "^([a-fA-F0-9]{64})\s\s(.+)$") {
            Fail "invalid checksum line: $line"
        }
        $expected = $Matches[1].ToLowerInvariant()
        $rel = $Matches[2]
        $target = Join-Path $ReleaseDir ($rel -replace "/", "\")
        Test-RequiredFile $target
        $actual = (Get-FileHash -LiteralPath $target -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            Fail "checksum mismatch for $rel"
        }
    }
}

function Assert-NoBundledMsi {
    param([string]$ReleaseDir)
    $msi = Get-ChildItem -LiteralPath $ReleaseDir -Recurse -File -Filter *.msi -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $msi) {
        Fail "MSI bundled by default: $($msi.FullName)"
    }
}

function Assert-NoObviousSecrets {
    param([string]$ReleaseDir)
    $patterns = @(
        "password\s*=",
        "access_token\s*=",
        "refresh_token\s*=",
        "Authorization:\s*Bearer\s+",
        "--invite\s+(?!`"<INVITE>`"|'<INVITE>'|<INVITE>|`"<REDACTED>`"|'<REDACTED>'|<REDACTED>)\S+",
        "invite\s*=\s*(?!`"<INVITE>`"|'<INVITE>'|<INVITE>|`"<REDACTED>`"|'<REDACTED>'|<REDACTED>)\S+"
    )
    $files = Get-ChildItem -LiteralPath $ReleaseDir -Recurse -File |
        Where-Object { $_.Extension -in @(".md", ".txt", ".ps1", ".json", ".yaml", ".yml") }
    foreach ($file in $files) {
        $text = Get-Content -LiteralPath $file.FullName -Raw
        foreach ($pattern in $patterns) {
            if ($text -match $pattern) {
                Fail "obvious sensitive value pattern found in $(Get-RelativePath -BasePath $ReleaseDir -Path $file.FullName): $pattern"
            }
        }
    }
}

function Assert-Manifest {
    param([string]$ReleaseDir, [bool]$RequireSigned)
    $manifestPath = Join-Path $ReleaseDir "RELEASE-MANIFEST.json"
    Test-RequiredFile $manifestPath
    try {
        $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
    } catch {
        Fail "RELEASE-MANIFEST.json is not valid JSON: $($_.Exception.Message)"
    }
    foreach ($field in @("product", "version", "commit", "build_date", "built_by", "go_version", "target", "artifacts", "security_notes")) {
        if ($null -eq $manifest.$field) { Fail "manifest missing field: $field" }
    }
    if ($manifest.security_notes.msi_bundled -ne $false) { Fail "manifest must record msi_bundled=false" }
    if ($manifest.security_notes.invite_included -ne $false) { Fail "manifest must record invite_included=false" }
    if ($manifest.security_notes.reports_included -ne $false) { Fail "manifest must record reports_included=false" }
    $artifacts = @($manifest.artifacts)
    if ($artifacts.Count -eq 0) { Fail "manifest contains no artifacts" }
    $cli = $artifacts | Where-Object { $_.path -eq "kigrepair.exe" } | Select-Object -First 1
    if ($null -eq $cli) { Fail "manifest missing kigrepair.exe artifact" }
    foreach ($artifact in $artifacts) {
        $path = Join-Path $ReleaseDir ($artifact.path -replace "/", "\")
        Test-RequiredFile $path
        $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($hash -ne [string]$artifact.sha256) {
            Fail "manifest checksum mismatch for $($artifact.path)"
        }
        if ($RequireSigned -and ($artifact.signed -ne $true -or $artifact.signature_verified -ne $true)) {
            Fail "required signed artifact is unsigned or unverified: $($artifact.path)"
        }
    }
}

function Assert-Signatures {
    param([string]$ReleaseDir, [bool]$RequireSigned)
    $exeFiles = Get-ChildItem -LiteralPath $ReleaseDir -File -Filter *.exe
    foreach ($exe in $exeFiles) {
        $signature = Get-AuthenticodeSignature -LiteralPath $exe.FullName
        if ($RequireSigned -and $signature.Status -ne "Valid") {
            Fail "signature verification failed for $($exe.Name): $($signature.Status)"
        }
    }
}

function Assert-Zip {
    param([string]$ZipPath)
    if ([string]::IsNullOrWhiteSpace($ZipPath)) { return }
    Test-RequiredFile $ZipPath
    $sidecar = "$ZipPath.sha256"
    if (Test-Path -LiteralPath $sidecar -PathType Leaf) {
        $line = Get-Content -LiteralPath $sidecar -Raw
        if ($line -notmatch "^([a-fA-F0-9]{64})\s\s(.+\.zip)\s*$") {
            Fail "invalid zip checksum sidecar"
        }
        $expected = $Matches[1].ToLowerInvariant()
        $actual = (Get-FileHash -LiteralPath $ZipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            Fail "zip checksum mismatch"
        }
    }
}

if ([string]::IsNullOrWhiteSpace($ReleaseDir)) {
    Fail "-ReleaseDir is required"
}
$ReleaseDir = (Resolve-Path $ReleaseDir).Path

foreach ($rel in @(
    "kigrepair.exe",
    "README.md",
    "SUPPORT-RUNBOOK.md",
    "docs/COMMAND-REFERENCE.md",
    "docs/SAFETY-MODEL.md",
    "docs/RELEASE-CHECKLIST.md",
    "docs/RELEASE-TRUST.md",
    "assets/README.txt",
    "examples/commands.ps1",
    "RELEASE-MANIFEST.json",
    "checksums.txt"
)) {
    Test-RequiredFile (Join-Path $ReleaseDir ($rel -replace "/", "\"))
}

Assert-NoBundledMsi -ReleaseDir $ReleaseDir
Assert-ChecksumFile -ReleaseDir $ReleaseDir
Assert-Manifest -ReleaseDir $ReleaseDir -RequireSigned ([bool]$RequireSigned)
Assert-Signatures -ReleaseDir $ReleaseDir -RequireSigned ([bool]$RequireSigned)
Assert-NoObviousSecrets -ReleaseDir $ReleaseDir
Assert-Zip -ZipPath $ZipPath

Write-Host "Release validation passed: $ReleaseDir"
