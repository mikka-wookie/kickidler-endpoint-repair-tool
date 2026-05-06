# VM Test Matrix

## Tier 0 - Local Developer Smoke

Purpose:

- Fast check before committing or releasing.
- Mostly read-only.
- No destructive repair required.

Required OS:

- Windows 10 x64 or Windows 11 x64.

Required commands:

```powershell
.\kigrepair.exe version --json
.\kigrepair.exe config show
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe cleanup --dry-run
.\kigrepair.exe collect-report
.\kigrepair.exe reports list
```

Recommended script:

```powershell
.\scripts\vm-tests\run-smoke-readonly.ps1 `
  -KigrepairPath ".\kigrepair.exe" `
  -OutDir ".\evidence\tier0-smoke" `
  -Profile standard
```

Pass criteria:

- Commands complete with expected success, warning, or validation exit codes documented for the current VM state.
- Evidence folder contains command output and `vm-smoke-result.json`.
- No raw invite or secret is present in reports or evidence.
- No product cleanup, install, uninstall, Defender mutation, service mutation, process termination, or file deletion occurs.

## Tier 1 - MVP Release Gate

Purpose:

- Minimum required validation before pilot/internal MVP.
- Proves core support workflows on real Windows states.

Required environments:

- Windows 10 x64, Defender enabled.
- Windows 11 x64, Defender enabled.
- One machine with Grabber not installed.
- One machine with healthy standard Grabber install.
- One machine with simulated or real broken service binary missing.
- One machine with partial MSI leftovers.
- One machine with hidden WMI mode, if available.

Required scenario categories:

- Clean not-installed detection.
- Healthy standard install detection and verification.
- Invalid installer blocks repair readiness.
- Missing invite blocks preflight/repair readiness.
- Non-admin blocks real repair.
- Normal Windows process false-positive protection.
- Support bundle and evidence redaction with fake invite.
- Repair dry-run plan generation using approved installer and env-var invite.

## Tier 2 - Expanded Regression

Purpose:

- Broader validation before internal-stable or customer-facing planning.
- Exercises policy, safety, Defender, GUI, and path mismatch cases.

Include where available:

- Defender disabled.
- Defender policy-managed.
- Third-party AV installed.
- Non-admin user.
- Admin user.
- Program Files standard path.
- Program Files x86 path.
- Helper path.
- Hidden WMI path.
- Service path mismatch.
- Process path mismatch.
- Missing installer.
- Invalid installer.
- Unsupported MSI filename.
- Stale reports retention.
- GUI MVP smoke if GUI is distributed.

Pass criteria:

- Pilot gates remain passing.
- Expanded cases are passed or explicitly waived with reason, owner, and release impact.
- Destructive cases show snapshot taken before mutation and restored after test when required.

