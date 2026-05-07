param(
    [Parameter(Mandatory = $true)]
    [string]$KigrepairPath,
    [string]$GuiPath = "",
    [string]$InstallerPath = "",
    [Parameter(Mandatory = $true)]
    [string]$OutDir,
    [string]$InviteFromEnvVar = "KIGREPAIR_TEST_INVITE",
    [switch]$SkipGui
)

$ErrorActionPreference = "Stop"

function Redact-Text {
    param([string]$Text, [string]$Secret)
    if ([string]::IsNullOrEmpty($Text)) { return "" }
    $out = $Text
    if (-not [string]::IsNullOrWhiteSpace($Secret)) {
        $out = $out.Replace($Secret, "<REDACTED>")
    }
    $out = $out -replace "Authorization:\s*Bearer\s+\S+", "Authorization: Bearer <REDACTED>"
    $out = $out -replace "Authorization:\s*Basic\s+\S+", "Authorization: Basic <REDACTED>"
    return $out
}

function Invoke-SmokeCommand {
    param(
        [string]$Name,
        [string[]]$CommandArgs,
        [string]$FailureSeverity,
        [string]$Secret
    )
    $stdoutFile = Join-Path $OutDir "$Name.stdout.txt"
    $stderrFile = Join-Path $OutDir "$Name.stderr.txt"
    $started = (Get-Date).ToUniversalTime()
    $output = ""
    $exitCode = 0
    try {
        $output = & $KigrepairPath @CommandArgs 2>&1 | Out-String
        $exitCode = $LASTEXITCODE
        if ($null -eq $exitCode) { $exitCode = 0 }
    } catch {
        $output = $_.Exception.Message
        $exitCode = 1
    }
    $finished = (Get-Date).ToUniversalTime()
    $redacted = Redact-Text -Text $output -Secret $Secret
    Set-Content -LiteralPath $stdoutFile -Encoding UTF8 -Value $redacted
    Set-Content -LiteralPath $stderrFile -Encoding UTF8 -Value ""
    $reportPaths = @()
    foreach ($match in [regex]::Matches($redacted, "Report directory:\s*(.+)")) {
        $reportPaths += $match.Groups[1].Value.Trim()
    }
    return [ordered]@{
        name = $Name
        redacted_command = ".\kigrepair.exe " + (($CommandArgs | ForEach-Object { if ($_ -eq $Secret) { "<REDACTED>" } else { $_ } }) -join " ")
        exit_code = [int]$exitCode
        failure_severity = $FailureSeverity
        started_at = $started.ToString("o")
        finished_at = $finished.ToString("o")
        stdout_path = $stdoutFile
        stderr_path = $stderrFile
        report_paths = $reportPaths
    }
}

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
$KigrepairPath = (Resolve-Path -LiteralPath $KigrepairPath).Path
$invite = [Environment]::GetEnvironmentVariable($InviteFromEnvVar)

$commands = @()
$commands += Invoke-SmokeCommand -Name "version-json" -CommandArgs @("version", "--json") -FailureSeverity "P1" -Secret $invite
$commands += Invoke-SmokeCommand -Name "config-show" -CommandArgs @("config", "show") -FailureSeverity "P1" -Secret $invite
$commands += Invoke-SmokeCommand -Name "check" -CommandArgs @("check", "--output", $OutDir) -FailureSeverity "P1" -Secret $invite
$commands += Invoke-SmokeCommand -Name "verify" -CommandArgs @("verify", "--output", $OutDir) -FailureSeverity "P1" -Secret $invite
$commands += Invoke-SmokeCommand -Name "cleanup-dry-run" -CommandArgs @("cleanup", "--dry-run", "--output", $OutDir) -FailureSeverity "P1" -Secret $invite
$commands += Invoke-SmokeCommand -Name "collect-report" -CommandArgs @("collect-report", "--output", $OutDir) -FailureSeverity "P1" -Secret $invite
$commands += Invoke-SmokeCommand -Name "reports-list" -CommandArgs @("reports", "list", "--output", $OutDir) -FailureSeverity "P2" -Secret $invite

if (-not [string]::IsNullOrWhiteSpace($InstallerPath) -and -not [string]::IsNullOrWhiteSpace($invite)) {
    $commands += Invoke-SmokeCommand -Name "preflight" -CommandArgs @("preflight", "--installer", $InstallerPath, "--invite", $invite, "--output", $OutDir) -FailureSeverity "P1" -Secret $invite
    $commands += Invoke-SmokeCommand -Name "repair-dry-run" -CommandArgs @("repair", "--dry-run", "--installer", $InstallerPath, "--invite", $invite, "--output", $OutDir) -FailureSeverity "P1" -Secret $invite
}

$guiStatus = "skipped"
if (-not $SkipGui -and -not [string]::IsNullOrWhiteSpace($GuiPath)) {
    $guiStatus = $(if (Test-Path -LiteralPath $GuiPath -PathType Leaf) { "present_manual_check_required" } else { "missing" })
}

$failed = @($commands | Where-Object { $_.exit_code -ne 0 })
$result = [ordered]@{
    status = $(if ($failed.Count -eq 0 -and $guiStatus -ne "missing") { "passed" } else { "failed" })
    generated_at = (Get-Date).ToUniversalTime().ToString("o")
    kigrepair_path = $KigrepairPath
    gui_path = $(if ([string]::IsNullOrWhiteSpace($GuiPath)) { "" } else { $GuiPath })
    gui_status = $guiStatus
    invite_env_var = $InviteFromEnvVar
    invite_present = (-not [string]::IsNullOrWhiteSpace($invite))
    commands = $commands
}

$jsonPath = Join-Path $OutDir "smoke-result.json"
$summaryPath = Join-Path $OutDir "smoke-summary.md"
$result | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $jsonPath -Encoding UTF8

$summary = @(
    "# MVP Smoke Summary",
    "",
    "Status: $($result.status)",
    "Generated: $($result.generated_at)",
    "GUI status: $guiStatus",
    "",
    "| Command | Exit code | Severity candidate |",
    "|---|---:|---|"
)
foreach ($cmd in $commands) {
    $summary += "| $($cmd.name) | $($cmd.exit_code) | $($cmd.failure_severity) |"
}
Set-Content -LiteralPath $summaryPath -Encoding UTF8 -Value $summary

Write-Host "Smoke result: $jsonPath"
Write-Host "Smoke summary: $summaryPath"
Write-Host "Status: $($result.status)"
if ($result.status -ne "passed") {
    exit 1
}
