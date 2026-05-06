# Scenario Setup Templates

These files are guarded VM setup templates for manually creating risk scenarios. They are intentionally conservative.

Rules:

- Disposable VM snapshots only.
- Elevated PowerShell where required.
- Explicit `-IUnderstandThisIsDestructive` required for any mutation-capable template.
- Prefer `-DryRun`.
- Touch only exact known Kickidler/Grabber allowlisted paths and keys.
- Never modify normal Windows `svchost.exe`, `WmiPrvSE.exe`, or `RuntimeBroker.exe`.
- Never delete arbitrary Windows paths.
- Do not store invite values in setup scripts.

Current templates are documentation-first and do not perform product path deletion. If a scenario needs mutation, review the exact planned actions and perform the setup manually in a disposable VM or extend the template in a separate reviewed change.

