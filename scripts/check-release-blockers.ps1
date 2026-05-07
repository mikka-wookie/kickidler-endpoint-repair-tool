param(
    [string]$ReleaseBlockers = ".\docs\RELEASE-BLOCKERS.md",
    [string]$BugBurndown = ".\docs\BUG-BURNDOWN.md",
    [string]$BlockersJson = ".\docs\release-blockers.json",
    [string]$SmokeResult = "",
    [string]$VMResult = ""
)

$ErrorActionPreference = "Stop"

function Add-Blocker {
    param([string]$Id, [string]$Severity, [string]$Status, [string]$Title, [string]$Source)
    if ([string]::IsNullOrWhiteSpace($Severity)) { return }
    $script:Items += [ordered]@{
        id = $Id
        severity = $Severity.ToUpperInvariant()
        status = $Status.ToLowerInvariant()
        title = $Title
        source = $Source
    }
}

function Read-JsonFile {
    param([string]$Path)
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Parse-BugTable {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return }
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -notmatch "^\|\s*BUG-" -or $line -match "\|---") { continue }
        $parts = $line.Trim("|").Split("|") | ForEach-Object { $_.Trim() }
        if ($parts.Count -lt 5) { continue }
        Add-Blocker -Id $parts[0] -Severity $parts[1] -Status $parts[4] -Title $parts[3] -Source $Path
    }
}

function Parse-Markers {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return }
    $id = ""
    $severity = ""
    $status = ""
    $title = ""
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -match "^Status:\s*(.+)$") { $status = $Matches[1].Trim(); continue }
        if ($line -match "^Severity:\s*(P[0-3]).*$") { $severity = $Matches[1].Trim(); continue }
        if ($line -match "^ID:\s*(.+)$") { $id = $Matches[1].Trim(); continue }
        if ($line -match "^Title:\s*(.+)$") {
            $title = $Matches[1].Trim()
            Add-Blocker -Id $id -Severity $severity -Status $status -Title $title -Source $Path
            $id = ""; $severity = ""; $status = ""; $title = ""
        }
    }
}

function Add-SmokeFailures {
    param([string]$Path)
    $smoke = Read-JsonFile $Path
    if ($null -eq $smoke) { return }
    foreach ($cmd in @($smoke.commands)) {
        if ($null -eq $cmd) { continue }
        if ([int]$cmd.exit_code -ne 0) {
            Add-Blocker -Id "SMOKE-$($cmd.name)" -Severity ([string]$cmd.failure_severity) -Status "open" -Title "Smoke command failed: $($cmd.name)" -Source $Path
        }
    }
    if ([string]$smoke.status -eq "failed") {
        Add-Blocker -Id "SMOKE-OVERALL" -Severity "P1" -Status "open" -Title "Smoke run failed" -Source $Path
    }
}

$script:Items = @()

if (-not (Test-Path -LiteralPath $ReleaseBlockers -PathType Leaf)) {
    throw "Release blocker document missing: $ReleaseBlockers"
}

Parse-Markers -Path $ReleaseBlockers
Parse-BugTable -Path $BugBurndown

$json = Read-JsonFile $BlockersJson
if ($null -ne $json) {
    foreach ($item in @($json.blockers)) {
        Add-Blocker -Id ([string]$item.id) -Severity ([string]$item.severity) -Status ([string]$item.status) -Title ([string]$item.title) -Source $BlockersJson
    }
}

Add-SmokeFailures -Path $SmokeResult
Add-SmokeFailures -Path $VMResult

$open = @($Items | Where-Object { $_.status -notin @("closed", "fixed", "resolved", "done") })
$p0 = @($open | Where-Object { $_.severity -eq "P0" })
$p1 = @($open | Where-Object { $_.severity -eq "P1" })
$p2 = @($open | Where-Object { $_.severity -eq "P2" })

$result = [ordered]@{
    status = $(if ($p0.Count -gt 0 -or $p1.Count -gt 0) { "failed" } elseif ($p2.Count -gt 0) { "warning" } else { "passed" })
    open_p0 = $p0.Count
    open_p1 = $p1.Count
    open_p2 = $p2.Count
    open = $open
}

$result | ConvertTo-Json -Depth 6

if ($p0.Count -gt 0 -or $p1.Count -gt 0) {
    exit 1
}
if ($p2.Count -gt 0) {
    Write-Warning "Open P2 issues remain."
}
