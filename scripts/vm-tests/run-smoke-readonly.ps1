param(
    [Parameter(Mandatory = $true)][string]$KigrepairPath,
    [Parameter(Mandatory = $true)][string]$OutDir,
    [ValidateSet("standard", "conservative", "diagnostic")][string]$Profile = "standard"
)

$ErrorActionPreference = "Stop"

function Resolve-Executable {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "kigrepair executable not found: $Path"
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

function Invoke-KigrepairCommand {
    param(
        [string]$Exe,
        [string[]]$Arguments,
        [string]$Name,
        [string]$OutputDir
    )
    $stdoutPath = Join-Path $OutputDir "$Name.stdout.txt"
    $stderrPath = Join-Path $OutputDir "$Name.stderr.txt"
    $argumentLine = Join-Arguments -Arguments $Arguments
    $process = Start-Process -FilePath $Exe -ArgumentList $argumentLine -NoNewWindow -Wait -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
    return [ordered]@{
        name = $Name
        command = "kigrepair " + ($Arguments -join " ")
        exit_code = $process.ExitCode
        stdout = $stdoutPath
        stderr = $stderrPath
    }
}

$exe = Resolve-Executable -Path $KigrepairPath
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$out = (Resolve-Path -LiteralPath $OutDir).Path
$commandOut = Join-Path $out "command-output"
New-Item -ItemType Directory -Force -Path $commandOut | Out-Null

$commands = @(
    @{ Name = "01-version-json"; Args = @("version", "--json") },
    @{ Name = "02-config-show"; Args = @("config", "show", "--profile", $Profile) },
    @{ Name = "03-check"; Args = @("check", "--profile", $Profile) },
    @{ Name = "04-verify"; Args = @("verify", "--profile", $Profile) },
    @{ Name = "05-cleanup-dry-run"; Args = @("cleanup", "--dry-run", "--profile", $Profile) },
    @{ Name = "06-collect-report"; Args = @("collect-report", "--profile", $Profile) },
    @{ Name = "07-reports-list"; Args = @("reports", "list", "--profile", $Profile) }
)

$results = @()
foreach ($cmd in $commands) {
    Write-Host "Running read-only smoke command: $($cmd.Name)"
    $results += Invoke-KigrepairCommand -Exe $exe -Arguments $cmd.Args -Name $cmd.Name -OutputDir $commandOut
}

$failed = @($results | Where-Object { $_.exit_code -ge 8 })
$status = if ($failed.Count -gt 0) { "failed" } else { "passed" }

$summary = [ordered]@{
    schema = "kigrepair.vm_smoke_result.v1"
    created_at = (Get-Date).ToString("o")
    profile = $Profile
    kigrepair_path = $exe
    output_dir = $out
    status = $status
    destructive = $false
    invite_required = $false
    installer_required = $false
    commands = $results
}

$summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $out "vm-smoke-result.json") -Encoding UTF8
if ($status -ne "passed") {
    Write-Error "VM smoke failed. See $out"
    exit 1
}
Write-Host "VM smoke completed: $out"

