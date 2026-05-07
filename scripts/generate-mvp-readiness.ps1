param(
    [string]$SmokeResult = "",
    [string]$VMSummary = "",
    [string]$BugBurndown = ".\docs\BUG-BURNDOWN.md",
    [string]$ReleaseManifest = "",
    [string]$OutFile = ".\docs\MVP-READINESS.md",
    [string]$Channel = "internal-mvp",
    [string]$GuiSmokeStatus = "manual_not_run"
)

$ErrorActionPreference = "Stop"

function Read-JsonFile {
    param([string]$Path)
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Count-OpenBugs {
    param([string]$Path)
    $counts = @{ P0 = 0; P1 = 0; P2 = 0; P3 = 0 }
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $counts }
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -notmatch "^\|\s*BUG-" -or $line -match "\|---") { continue }
        $parts = $line.Trim("|").Split("|") | ForEach-Object { $_.Trim() }
        if ($parts.Count -lt 5) { continue }
        $severity = $parts[1].ToUpperInvariant()
        $status = $parts[4].ToLowerInvariant()
        if ($status -in @("closed", "fixed", "resolved", "done")) { continue }
        if ($counts.ContainsKey($severity)) { $counts[$severity]++ }
    }
    return $counts
}

function Get-GitValue {
    param([string[]]$GitArgs, [string]$Fallback)
    try {
        $value = (& git @GitArgs 2>$null).Trim()
        if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
    } catch {}
    return $Fallback
}

$smoke = Read-JsonFile $SmokeResult
$manifest = Read-JsonFile $ReleaseManifest
$counts = Count-OpenBugs -Path $BugBurndown

$version = "dev"
$commit = Get-GitValue -GitArgs @("rev-parse", "--short", "HEAD") -Fallback "unknown"
$buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
if ($null -ne $manifest) {
    if ($manifest.version) { $version = [string]$manifest.version }
    if ($manifest.commit) { $commit = [string]$manifest.commit }
    if ($manifest.build_date) { $buildDate = [string]$manifest.build_date }
}

$cliSmokeStatus = "not_run"
if ($null -ne $smoke -and $smoke.status) {
    $cliSmokeStatus = [string]$smoke.status
}

$vmStatus = "not_run"
if (-not [string]::IsNullOrWhiteSpace($VMSummary) -and (Test-Path -LiteralPath $VMSummary)) {
    $vmStatus = "provided"
}

$recommendation = "NO-GO"
if ($counts.P0 -gt 0 -or $counts.P1 -gt 0) {
    $recommendation = "NO-GO"
} elseif ($cliSmokeStatus -ne "passed") {
    $recommendation = "NO-GO"
} elseif ($counts.P2 -gt 5) {
    $recommendation = "PILOT ONLY"
} else {
    $recommendation = "PILOT CANDIDATE"
}

$requiredFixes = @()
if ($counts.P0 -gt 0) { $requiredFixes += "Close all open P0 blockers." }
if ($counts.P1 -gt 0) { $requiredFixes += "Close all open P1 blockers." }
if ($cliSmokeStatus -ne "passed") { $requiredFixes += "Run and pass CLI smoke." }
if ($requiredFixes.Count -eq 0) { $requiredFixes += "None for pilot candidate gate." }

$evidence = @()
if (-not [string]::IsNullOrWhiteSpace($SmokeResult)) { $evidence += $SmokeResult }
if (-not [string]::IsNullOrWhiteSpace($VMSummary)) { $evidence += $VMSummary }
if (-not [string]::IsNullOrWhiteSpace($ReleaseManifest)) { $evidence += $ReleaseManifest }
if ($evidence.Count -eq 0) { $evidence += "pending" }

$lines = @(
    "# MVP Readiness",
    "",
    "Version: $version",
    "Commit: $commit",
    "Build date: $buildDate",
    "Channel: $Channel",
    "CLI smoke status: $cliSmokeStatus",
    "GUI smoke status: $GuiSmokeStatus",
    "VM validation status: $vmStatus",
    "P0 open: $($counts.P0)",
    "P1 open: $($counts.P1)",
    "P2 open: $($counts.P2)",
    "Release recommendation: $recommendation",
    "",
    "## Required Fixes"
)
foreach ($item in $requiredFixes) { $lines += "- $item" }
$lines += @(
    "",
    "## Known Limitations",
    "",
    "See [Known Limitations](KNOWN-LIMITATIONS.md).",
    "",
    "## Evidence Paths"
)
foreach ($item in $evidence) { $lines += "- $item" }

$parent = Split-Path -Parent $OutFile
if (-not [string]::IsNullOrWhiteSpace($parent)) {
    New-Item -ItemType Directory -Path $parent -Force | Out-Null
}
Set-Content -LiteralPath $OutFile -Encoding UTF8 -Value $lines
Write-Host "MVP readiness report written: $OutFile"
Write-Host "Release recommendation: $recommendation"
