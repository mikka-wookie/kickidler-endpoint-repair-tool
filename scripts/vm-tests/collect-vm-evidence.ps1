param(
    [string]$ScenarioId = "manual",
    [Parameter(Mandatory = $true)][string]$OutDir,
    [Parameter(Mandatory = $true)][string]$KigrepairPath,
    [string]$ReleaseDir = ""
)

$ErrorActionPreference = "Stop"

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Path -LiteralPath $KigrepairPath -PathType Leaf)) {
    throw "kigrepair executable not found: $KigrepairPath"
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$out = (Resolve-Path -LiteralPath $OutDir).Path
$commandOut = Join-Path $out "command-output"
$reportsOut = Join-Path $out "reports"
$screenshotsOut = Join-Path $out "screenshots"
New-Item -ItemType Directory -Force -Path $commandOut, $reportsOut, $screenshotsOut | Out-Null

$exe = (Resolve-Path -LiteralPath $KigrepairPath).Path
& $exe version --json > (Join-Path $commandOut "version.json") 2> (Join-Path $commandOut "version.stderr.txt")
$versionExit = $LASTEXITCODE

$os = Get-CimInstance -ClassName Win32_OperatingSystem | Select-Object Caption, Version, BuildNumber, OSArchitecture
$defender = $null
try {
    $defender = Get-MpComputerStatus | Select-Object AMServiceEnabled, AntivirusEnabled, RealTimeProtectionEnabled, IsTamperProtected, NISEnabled
} catch {
    $defender = [ordered]@{ unavailable = $true; error = $_.Exception.Message }
}

$reportRoot = "C:\ProgramData\kigrepair\Reports"
$reportDirs = @()
$latestCopied = $null
if (Test-Path -LiteralPath $reportRoot -PathType Container) {
    $reportDirs = @(Get-ChildItem -LiteralPath $reportRoot -Directory -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending)
    $latest = $reportDirs | Select-Object -First 1
    if ($null -ne $latest) {
        $dest = Join-Path $reportsOut $latest.Name
        Copy-Item -LiteralPath $latest.FullName -Destination $dest -Recurse -Force
        $latestCopied = $dest
    }
}

if (-not [string]::IsNullOrWhiteSpace($ReleaseDir) -and (Test-Path -LiteralPath $ReleaseDir -PathType Container)) {
    foreach ($name in @("RELEASE-MANIFEST.json", "checksums.txt", "SIGNATURES.txt")) {
        $src = Join-Path $ReleaseDir $name
        if (Test-Path -LiteralPath $src -PathType Leaf) {
            Copy-Item -LiteralPath $src -Destination (Join-Path $out $name) -Force
        }
    }
}

"Place manually approved screenshots for this scenario in this directory. Do not include raw invite values." |
    Set-Content -LiteralPath (Join-Path $screenshotsOut "README.txt") -Encoding UTF8

$supportBundles = @()
if ($null -ne $latestCopied) {
    $supportBundles = @(Get-ChildItem -LiteralPath $latestCopied -Recurse -File -Filter "kigrepair-support-bundle.zip" -ErrorAction SilentlyContinue | ForEach-Object { $_.FullName })
}

$manifest = [ordered]@{
    schema = "kigrepair.vm_evidence_manifest.v1"
    created_at = (Get-Date).ToString("o")
    scenario_id = $ScenarioId
    output_dir = $out
    kigrepair_path = $exe
    version_exit_code = $versionExit
    os = $os
    is_elevated = Test-IsAdmin
    defender = $defender
    report_root = $reportRoot
    report_dirs = @($reportDirs | Select-Object -First 20 | ForEach-Object { $_.FullName })
    latest_report_copied = $latestCopied
    support_bundle_paths = $supportBundles
    excluded_collection = @("raw invite", "unrelated customer files", "full user profile", "browser history", "arbitrary ProgramData contents", "installer MSI files")
}

$manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $out "evidence-manifest.json") -Encoding UTF8
Write-Host "VM evidence collected: $out"

