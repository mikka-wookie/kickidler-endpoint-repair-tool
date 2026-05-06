# kigrepair VM Testing

This directory defines the real-Windows VM validation process for `kigrepair` releases.

Use this framework before internal support MVP release and before promoting builds to broader internal use. It converts fake-provider and support-risk scenarios into repeatable VM checks, smoke scripts, evidence collection, and release gates.

## Documents

- [VM Matrix](VM-MATRIX.md): validation tiers and required environments.
- [VM Setup](VM-SETUP.md): VM layout, snapshots, local assets, and safety rules.
- [VM Scenarios](VM-SCENARIOS.md): fake-provider to VM scenario mapping.
- [VM Smoke Tests](VM-SMOKE-TESTS.md): read-only and GUI smoke procedures.
- [VM Destructive Tests](VM-DESTRUCTIVE-TESTS.md): guarded destructive test policy.
- [Evidence Checklist](VM-EVIDENCE-CHECKLIST.md): required artifacts and exclusions.
- [Release Gates](VM-RELEASE-GATES.md): pilot and internal-stable gates.
- [Results Template](VM-RESULTS-TEMPLATE.md): per-scenario result record.
- [Issue Template](VM-ISSUE-TEMPLATE.md): VM validation defect record.

## Safety Rules

- Run destructive VM tests only on disposable snapshots.
- Do not run destructive tests on production or customer endpoints.
- Do not commit Grabber MSI files.
- Do not commit or paste real invite values.
- Use `<INVITE>`, `<REDACTED>`, or a local environment variable for invite input.
- Do not collect unrelated customer files, user profiles, browser history, or arbitrary `ProgramData` contents.
- Do not weaken runtime gates for admin, `--yes`, rollback, policy, installer validation, Defender validation, service/process/path trust, or redaction.

## Script Location

VM scripts live under:

```text
scripts\vm-tests\
```

Read-only scripts are safe for local smoke runs except for normal report/evidence writes. Destructive scripts are templates and require explicit guard parameters.

