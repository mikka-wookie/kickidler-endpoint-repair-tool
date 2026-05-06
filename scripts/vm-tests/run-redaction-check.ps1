param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string]$SecretPattern
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $Path)) {
    throw "Path not found: $Path"
}

$root = (Resolve-Path -LiteralPath $Path).Path
$patterns = @(
    [regex]::Escape($SecretPattern),
    "password\s*=",
    "access_token\s*=",
    "refresh_token\s*=",
    "Authorization:\s*Bearer",
    "Authorization:\s*Basic"
)

$textExtensions = @(".txt", ".log", ".json", ".jsonl", ".md", ".xml", ".yaml", ".yml", ".csv", ".ps1")
$matches = @()
$files = Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue |
    Where-Object { $_.Extension -in $textExtensions }

foreach ($file in $files) {
    $content = Get-Content -LiteralPath $file.FullName -Raw -ErrorAction SilentlyContinue
    if ($null -eq $content) { continue }
    foreach ($pattern in $patterns) {
        if ($content -match $pattern) {
            $matches += [ordered]@{
                file = $file.FullName
                pattern = if ($pattern -eq [regex]::Escape($SecretPattern)) { "provided-secret-pattern" } else { $pattern }
            }
        }
    }
}

$zipFiles = @(Get-ChildItem -LiteralPath $root -Recurse -File -Filter "*.zip" -ErrorAction SilentlyContinue)
$zipTempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("kigrepair-redaction-zip-" + [guid]::NewGuid().ToString("N"))
try {
    foreach ($zip in $zipFiles) {
        $zipOut = Join-Path $zipTempRoot ([System.IO.Path]::GetFileNameWithoutExtension($zip.Name) + "-" + [guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Force -Path $zipOut | Out-Null
        try {
            Expand-Archive -LiteralPath $zip.FullName -DestinationPath $zipOut -Force
        } catch {
            $matches += [ordered]@{
                file = $zip.FullName
                pattern = "zip-expand-failed: $($_.Exception.Message)"
            }
            continue
        }
        $zipTextFiles = Get-ChildItem -LiteralPath $zipOut -Recurse -File -ErrorAction SilentlyContinue |
            Where-Object { $_.Extension -in $textExtensions }
        foreach ($file in $zipTextFiles) {
            $content = Get-Content -LiteralPath $file.FullName -Raw -ErrorAction SilentlyContinue
            if ($null -eq $content) { continue }
            foreach ($pattern in $patterns) {
                if ($content -match $pattern) {
                    $matches += [ordered]@{
                        file = "$($zip.FullName)!$($file.FullName.Substring($zipOut.Length).TrimStart('\'))"
                        pattern = if ($pattern -eq [regex]::Escape($SecretPattern)) { "provided-secret-pattern" } else { $pattern }
                    }
                }
            }
        }
    }
} finally {
    if (Test-Path -LiteralPath $zipTempRoot) {
        Remove-Item -LiteralPath $zipTempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}

$result = [ordered]@{
    schema = "kigrepair.vm_redaction_check.v1"
    created_at = (Get-Date).ToString("o")
    path = $root
    checked_files = @($files).Count
    checked_zip_files = @($zipFiles).Count
    status = if (@($matches).Count -eq 0) { "passed" } else { "failed" }
    matches = $matches
}

$resultPath = Join-Path $root "redaction-check-result.json"
$result | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $resultPath -Encoding UTF8
if (@($matches).Count -gt 0) {
    Write-Error "Redaction check failed. See $resultPath"
    exit 1
}
Write-Host "Redaction check passed: $root"
