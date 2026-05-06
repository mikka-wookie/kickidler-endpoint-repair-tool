# VM Destructive Tests

Destructive VM tests are for disposable snapshots only. They may modify services, processes, files, registry keys, Defender exclusions, MSI state, or report retention state through `kigrepair` workflows.

## Required Guardrails

- Restore or create a disposable snapshot before the test.
- Use an elevated PowerShell session.
- Use an approved installer copied manually into `.\assets\`.
- Supply invite through a local environment variable only.
- Pass explicit destructive guard parameters such as `-IUnderstandThisIsDestructive`.
- Keep command outputs and evidence redacted.
- Verify final state with `verify` and `collect-report`.
- Restore the snapshot after the test unless the result needs to be preserved for escalation.

## Guarded Repair Template

```powershell
$env:KIGREPAIR_TEST_INVITE = "<INVITE>"

.\scripts\vm-tests\run-destructive-repair-template.ps1 `
  -KigrepairPath ".\kigrepair.exe" `
  -InstallerPath ".\assets\grabberEM.x64.msi" `
  -OutDir ".\evidence\real-repair" `
  -Profile conservative `
  -IUnderstandThisIsDestructive
```

The template runs:

- `preflight`
- `repair --dry-run`
- `repair --yes`
- `verify`
- `collect-report`

## Blockers

Stop and fail the release gate if any destructive test shows:

- Mutation outside allowlisted paths.
- Service/process mutation on path mismatch.
- Normal Windows process treated as Grabber.
- Real repair running without rollback snapshot.
- Destructive action running without admin.
- Non-interactive destructive action running without `--yes`.
- Raw invite or secret in reports, support bundle, screenshots, tickets, or evidence.

## Scenario Setup Templates

Setup templates under `scripts\vm-tests\scenario-setup\` are intentionally guarded. If the setup risk is high, keep the template documentation-only and create the VM state manually from an approved snapshot.

