param(
    [Parameter(Mandatory = $true)][string]$KigrepairPath,
    [Parameter(Mandatory = $true)][string]$InstallerPath,
    [Parameter(Mandatory = $true)][string]$OutDir,
    [ValidateSet("standard", "conservative", "diagnostic")][string]$Profile = "standard",
    [string]$InviteFromEnvVar = "KIGREPAIR_TEST_INVITE"
)

$ErrorActionPreference = "Stop"

function Resolve-RequiredFile {
    param([string]$Path, [string]$Label)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label not found: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
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
    $argumentLine = Join-Arguments -Arguments $Arguments
    $process = Start-Process -FilePath $Exe -ArgumentList $argumentLine -NoNewWindow -Wait -PassThru -RedirectStandardOutput $rawStdout -RedirectStandardError $rawStderr
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

$invite = [Environment]::GetEnvironmentVariable($InviteFromEnvVar)
if ([string]::IsNullOrWhiteSpace($invite)) {
    Write-Error "Invite environment variable is missing. Set `$env:$InviteFromEnvVar for this local VM session. Do not put invite values in scripts or evidence."
    exit 1
}

$exe = Resolve-RequiredFile -Path $KigrepairPath -Label "kigrepair executable"
$installer = Resolve-RequiredFile -Path $InstallerPath -Label "installer"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$out = (Resolve-Path -LiteralPath $OutDir).Path
$commandOut = Join-Path $out "command-output"
New-Item -ItemType Directory -Force -Path $commandOut | Out-Null

$commands = @(
    @{ Name = "01-preflight"; Args = @("preflight", "--profile", $Profile, "--installer", $installer, "--invite", $invite) },
    @{ Name = "02-repair-dry-run"; Args = @("repair", "--dry-run", "--profile", $Profile, "--installer", $installer, "--invite", $invite) }
)

$results = @()
foreach ($cmd in $commands) {
    Write-Host "Running read-only command: $($cmd.Name)"
    $results += Invoke-KigrepairCommand -Exe $exe -Arguments $cmd.Args -Name $cmd.Name -OutputDir $commandOut -Secret $invite
}

$failed = @($results | Where-Object { $_.exit_code -ge 8 })
$status = if ($failed.Count -gt 0) { "failed" } else { "passed" }

$summary = [ordered]@{
    schema = "kigrepair.vm_preflight_dryrun_result.v1"
    created_at = (Get-Date).ToString("o")
    profile = $Profile
    kigrepair_path = $exe
    installer_filename = [System.IO.Path]::GetFileName($installer)
    output_dir = $out
    status = $status
    destructive = $false
    invite_required = $true
    invite_source_env_var = $InviteFromEnvVar
    commands = $results
}

$summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $out "vm-smoke-result.json") -Encoding UTF8
if ($status -ne "passed") {
    Write-Error "Preflight/dry-run validation failed. See $out"
    exit 1
}
Write-Host "Preflight/dry-run validation completed: $out"

