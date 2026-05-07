param(
    [Parameter(Mandatory = $true)]
    [string]$Path,
    [string]$SecretPattern = ""
)

$ErrorActionPreference = "Stop"

function Add-Finding {
    param([string]$File, [string]$Pattern, [string]$Message)
    $script:Findings += [ordered]@{
        file = $File
        pattern = $Pattern
        message = $Message
    }
}

if (-not (Test-Path -LiteralPath $Path)) {
    throw "Path not found: $Path"
}

$root = (Resolve-Path -LiteralPath $Path).Path
$script:Findings = @()
$allowedPlaceholders = @("<INVITE>", "<REDACTED>", "REDACTED")
$secretRegexes = @(
    @{ name = "bearer authorization"; pattern = "Authorization:\s*Bearer\s+(?!<REDACTED>|<TOKEN>|<INVITE>)[A-Za-z0-9._~+/=-]{12,}" },
    @{ name = "basic authorization"; pattern = "Authorization:\s*Basic\s+(?!<REDACTED>|<TOKEN>)[A-Za-z0-9+/=]{12,}" },
    @{ name = "invite assignment"; pattern = "(?i)\binvite\s*[=:]\s*(?!`"?<INVITE>`"?|`"?<REDACTED>`"?)[A-Za-z0-9][A-Za-z0-9._~+/=-]{8,}" },
    @{ name = "access token"; pattern = "(?i)\b(access_token|refresh_token|api_key)\s*[=:]\s*(?!`"?<REDACTED>`"?)[A-Za-z0-9._~+/=-]{12,}" }
)

$files = Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue |
    Where-Object { $_.Length -lt 20MB }

foreach ($file in $files) {
    try {
        $text = Get-Content -LiteralPath $file.FullName -Raw -ErrorAction Stop
    } catch {
        continue
    }
    if ($null -eq $text) { $text = "" }
    $rel = $file.FullName
    if ($root -ne $file.FullName) {
        try {
            $base = $root
            if (-not $base.EndsWith([System.IO.Path]::DirectorySeparatorChar)) {
                $base += [System.IO.Path]::DirectorySeparatorChar
            }
            $baseUri = [System.Uri]::new($base)
            $targetUri = [System.Uri]::new($file.FullName)
            $rel = [System.Uri]::UnescapeDataString($baseUri.MakeRelativeUri($targetUri).ToString()).Replace("/", "\")
        } catch {}
    }
    if (-not [string]::IsNullOrWhiteSpace($SecretPattern)) {
        if ($allowedPlaceholders -notcontains $SecretPattern -and $text.Contains($SecretPattern)) {
            Add-Finding -File $rel -Pattern "exact-secret" -Message "Exact secret pattern was found."
        }
    }
    foreach ($rule in $secretRegexes) {
        if ($text -match $rule.pattern) {
            Add-Finding -File $rel -Pattern $rule.name -Message "Potential raw secret pattern was found."
        }
    }
}

$result = [ordered]@{
    path = $root
    status = $(if ($script:Findings.Count -eq 0) { "passed" } else { "failed" })
    findings = $script:Findings
}

$result | ConvertTo-Json -Depth 5

if ($script:Findings.Count -gt 0) {
    exit 1
}
