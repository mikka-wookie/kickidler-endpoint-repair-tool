# Escalation Checklist

## Attach

- `kigrepair-support-bundle.zip`

## Include in ticket

- Customer/company identifier.
- OS version.
- Command executed, with invite replaced by `<INVITE>`.
- Exact user symptom.
- Classification primary issue.
- Recommendation primary action.
- Whether repair was executed.
- Whether admin rights were used.
- Installer filename.
- Installer SHA-256 if available.
- Final verification status.

## Do not include

- Raw invite value.
- Screenshots containing invite.
- Customer secrets.
- Unredacted shared logs.

## Invite Safety

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Escalation Triggers

- `unknown_install_state`
- `wmi_hidden_mode_inconsistent`
- `defender_status_unavailable` when exclusion state must be proven
- Repeated `verification_failed`
- Service path mismatch
- Process path mismatch
- Invalid MSI from an official source
- Rollback snapshot failure
- Repair modifies nothing but state remains broken
- Defender or third-party AV repeatedly removes binaries
- Cleanup plan rejects a path that support expected to be safe

## Before escalating

Run read-only collection:

```powershell
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe collect-report
```

Use real repair only when approved:

```powershell
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.
