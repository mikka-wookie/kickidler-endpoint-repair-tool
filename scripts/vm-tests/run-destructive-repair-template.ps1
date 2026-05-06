param(
    [Parameter(Mandatory = $true)][string]$KigrepairPath,
    [Parameter(Mandatory = $true)][string]$InstallerPath,
    [Parameter(Mandatory = $true)][string]$OutDir,
    [ValidateSet("standard", "conservative", "diagnostic")][string]$Profile = "standard",
    [string]$InviteFromEnvVar = "KIGREPAIR_TEST_INVITE",
    [switch]$IUnderstandThisIsDestructive
)

$ErrorActionPreference = "Stop"

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Join-Arguments {
    param([string[]]$Arguments)
    $escaped = foreach ($arg in $Arguments) {
        if ($arg -match '[\s"]') {
            '"' + ($arg -replace '"', '\"') + '"'
        } else {
            $arg
        }
    }
    return ($escaped -join " ")
}

function Redact-Text {
    param([string]$Text, [string]$Secret)
    if ([string]::IsNullOrEmpty($Text)) { return $Text }
    if (-not [string]::IsNullOrEmpty($Secret)) {
        $Text = $Text.Replace($Secret, "<REDACTED>")
    }
    return $Text
}

function Invoke-KigrepairCommand {
    param(
        [string]$Exe,
        [string[]]$Arguments,
        [string]$Name,
        [string]$OutputDir,
        [string]$Secret
    )
    $rawStdout = Join-Path $OutputDir "$Name.stdout.raw.tmp"
    $rawStderr = Join-Path $OutputDir "$Name.stderr.raw.tmp"
    $stdoutPath = Join-Path $OutputDir "$Name.stdout.txt"
    $stderrPath = Join-Path $OutputDir "$Name.stderr.txt"
    $process = Start-Process -FilePath $Exe -ArgumentList (Join-Arguments -Arguments $Arguments) -NoNewWindow -Wait -PassThru -RedirectStandardOutput $rawStdout -RedirectStandardError $rawStderr
    Redact-Text -Text (Get-Content -LiteralPath $rawStdout -Raw -ErrorAction SilentlyContinue) -Secret $Secret | Set-Content -LiteralPath $stdoutPath -Encoding UTF8
    Redact-Text -Text (Get-Content -LiteralPath $rawStderr -Raw -ErrorAction SilentlyContinue) -Secret $Secret | Set-Content -LiteralPath $stderrPath -Encoding UTF8
    Remove-Item -LiteralPath $rawStdout, $rawStderr -Force -ErrorAction SilentlyContinue
    $safeArgs = @($Arguments | ForEach-Object { if ($_ -eq $Secret) { "<REDACTED>" } else { $_ } })
    return [ordered]@{
        name = $Name
        command = "kigrepair " + ($safeArgs -join " ")
        exit_code = $process.ExitCode
        stdout = $stdoutPath
        stderr = $stderrPath
    }
}

if (-not $IUnderstandThisIsDestructive) {
    Write-Error "Refusing to run. This destructive VM template requires -IUnderstandThisIsDestructive."
    exit 1
}
if (-not (Test-IsAdmin)) {
    Write-Error "Refusing to run destructive repair template because this PowerShell session is not elevated."
    exit 1
}
$invite = [Environment]::GetEnvironmentVariable($InviteFromEnvVar)
if ([string]::IsNullOrWhiteSpace($invite)) {
    Write-Error "Invite environment variable is missing. Set `$env:$InviteFromEnvVar for this disposable VM session."
    exit 1
}
if (-not (Test-Path -LiteralPath $KigrepairPath -PathType Leaf)) { throw "kigrepair executable not found: $KigrepairPath" }
if (-not (Test-Path -LiteralPath $InstallerPath -PathType Leaf)) { throw "installer not found: $InstallerPath" }

Write-Warning "DESTRUCTIVE VM TEST: disposable VM only. Snapshot is required. This modifies system state through kigrepair repair --yes."
Write-Warning "Do not run on production or customer machines."

$exe = (Resolve-Path -LiteralPath $KigrepairPath).Path
$installer = (Resolve-Path -LiteralPath $InstallerPath).Path
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$out = (Resolve-Path -LiteralPath $OutDir).Path
$commandOut = Join-Path $out "command-output"
New-Item -ItemType Directory -Force -Path $commandOut | Out-Null

$commands = @(
    @{ Name = "01-preflight"; Args = @("preflight", "--profile", $Profile, "--installer", $installer, "--invite", $invite) },
    @{ Name = "02-repair-dry-run"; Args = @("repair", "--dry-run", "--profile", $Profile, "--installer", $installer, "--invite", $invite) },
    @{ Name = "03-repair-real"; Args = @("repair", "--profile", $Profile, "--installer", $installer, "--invite", $invite, "--yes") },
    @{ Name = "04-verify"; Args = @("verify", "--profile", $Profile) },
    @{ Name = "05-collect-report"; Args = @("collect-report", "--profile", $Profile) }
)

$results = @()
foreach ($cmd in $commands) {
    Write-Host "Running guarded destructive sequence step: $($cmd.Name)"
    $results += Invoke-KigrepairCommand -Exe $exe -Arguments $cmd.Args -Name $cmd.Name -OutputDir $commandOut -Secret $invite
}

$failed = @($results | Where-Object { $_.exit_code -ge 8 })
$status = if ($failed.Count -gt 0) { "failed" } else { "completed" }
$summary = [ordered]@{
    schema = "kigrepair.vm_destructive_repair_result.v1"
    created_at = (Get-Date).ToString("o")
    profile = $Profile
    output_dir = $out
    destructive = $true
    snapshot_required = $true
    invite_source_env_var = $InviteFromEnvVar
    status = $status
    commands = $results
}
$summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $out "vm-smoke-result.json") -Encoding UTF8
if ($status -eq "failed") {
    Write-Error "Guarded destructive repair sequence failed. See $out"
    exit 1
}
Write-Host "Guarded destructive repair sequence completed: $out"

