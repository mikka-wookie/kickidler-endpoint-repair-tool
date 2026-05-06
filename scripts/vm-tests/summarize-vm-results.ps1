param(
    [Parameter(Mandatory = $true)][string]$EvidenceRoot,
    [Parameter(Mandatory = $true)][string]$OutFile
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $EvidenceRoot -PathType Container)) {
    throw "Evidence root not found: $EvidenceRoot"
}

$root = (Resolve-Path -LiteralPath $EvidenceRoot).Path
$resultFiles = @(Get-ChildItem -LiteralPath $root -Recurse -File -Filter "vm-smoke-result.json" -ErrorAction SilentlyContinue)
$manifestFiles = @(Get-ChildItem -LiteralPath $root -Recurse -File -Filter "evidence-manifest.json" -ErrorAction SilentlyContinue)

$results = foreach ($file in $resultFiles) {
    $json = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
    [ordered]@{
        file = $file.FullName
        status = [string]$json.status
        profile = [string]$json.profile
        destructive = [bool]$json.destructive
        commands = @($json.commands | ForEach-Object {
            [ordered]@{ name = $_.name; exit_code = $_.exit_code }
        })
    }
}

$manifests = foreach ($file in $manifestFiles) {
    $json = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
    [ordered]@{
        file = $file.FullName
        scenario_id = [string]$json.scenario_id
        os = if ($null -ne $json.os) { [string]$json.os.Caption + " " + [string]$json.os.Version } else { "" }
        latest_report_copied = [string]$json.latest_report_copied
        support_bundle_paths = @($json.support_bundle_paths)
    }
}

$redactionFiles = @(Get-ChildItem -LiteralPath $root -Recurse -File -Filter "redaction-check-result.json" -ErrorAction SilentlyContinue)
$redaction = foreach ($file in $redactionFiles) {
    $json = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
    [ordered]@{ file = $file.FullName; status = [string]$json.status; checked_files = $json.checked_files }
}

$passed = @($results | Where-Object { $_.status -in @("passed", "completed") }).Count
$failed = @($results | Where-Object { $_.status -eq "failed" }).Count
$skipped = @($results | Where-Object { $_.status -eq "skipped" }).Count
$blocked = @($results | Where-Object { $_.status -eq "blocked" }).Count
$gateStatus = if ($failed -gt 0) { "fail" } elseif ($results.Count -eq 0) { "unknown" } else { "pass" }

$summary = [ordered]@{
    schema = "kigrepair.vm_test_summary.v1"
    created_at = (Get-Date).ToString("o")
    evidence_root = $root
    total_scenarios = $results.Count
    passed = $passed
    failed = $failed
    skipped = $skipped
    blocked = $blocked
    gate_status = $gateStatus
    os_versions = @($manifests | ForEach-Object { $_.os } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)
    scenario_ids = @($manifests | ForEach-Object { $_.scenario_id } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)
    report_dirs = @($manifests | ForEach-Object { $_.latest_report_copied } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    support_bundle_paths = @($manifests | ForEach-Object { $_.support_bundle_paths } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    redaction_checks = $redaction
    results = $results
    manifests = $manifests
}

$base = $OutFile
if ($base.EndsWith(".json", [System.StringComparison]::OrdinalIgnoreCase)) {
    $base = $base.Substring(0, $base.Length - 5)
}
if ($base.EndsWith(".md", [System.StringComparison]::OrdinalIgnoreCase)) {
    $base = $base.Substring(0, $base.Length - 3)
}
$jsonPath = "$base.json"
$mdPath = "$base.md"

$summary | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $jsonPath -Encoding UTF8

$md = @()
$md += "# VM Test Summary"
$md += ""
$md += "- Created: $($summary.created_at)"
$md += "- Evidence root: $root"
$md += "- Gate status: $gateStatus"
$md += "- Total scenarios: $($summary.total_scenarios)"
$md += "- Passed: $passed"
$md += "- Failed: $failed"
$md += "- Skipped: $skipped"
$md += "- Blocked: $blocked"
$md += ""
$md += "## Scenarios"
foreach ($manifest in $manifests) {
    $md += "- $($manifest.scenario_id): $($manifest.os)"
}
$md += ""
$md += "## Redaction Checks"
foreach ($check in $redaction) {
    $md += "- $($check.status): $($check.file)"
}
$md | Set-Content -LiteralPath $mdPath -Encoding UTF8

Write-Host "VM summary written: $jsonPath"
Write-Host "VM summary written: $mdPath"

