# Release Blockers

P0 and P1 issues block internal MVP release. This document defines the blocker classes that `scripts/check-release-blockers.ps1` enforces together with `docs/BUG-BURNDOWN.md` and `docs/release-blockers.json`.

## P0 Safety Blockers

- Raw invite leak anywhere.
- Destructive action without administrator rights.
- Destructive action without `--yes` or exact confirmation.
- Real repair without rollback snapshot.
- Cleanup target outside allowlist.
- Service/process mutation on path mismatch.
- Hidden WMI false positive that can mutate normal Windows files/processes.
- Unsupported MSI accepted for install/repair.
- GUI bypasses workflow service.
- GUI shells out to CLI for workflows.
- Support bundle includes secrets.

## P1 MVP Blockers

- GUI freezes or becomes Not Responding during normal workflows.
- GUI blank startup or long blocking initialization.
- Check, Verify, Preflight, or Repair Dry-Run fail unexpectedly.
- Missing installer/admin errors are not shown clearly.
- Report directory is not created.
- Operations/result JSON is invalid.
- Workflow status mapping is wrong.
- Primary issue/recommendation is missing from GUI after workflow.
- Repeated warnings flood GUI timeline.
- Open Report/Open Summary/Open Operations are broken.
- Collect-report fails without clear error.
- CLI command exit codes are inconsistent.

## P2 Support Usability Issues

- Ugly layout.
- Clipped labels/paths when copy still works.
- Non-critical wording issues.
- Missing convenience buttons.
- Incomplete docs.

## P3 Internal Cleanup

- Naming cleanup.
- Code duplication.
- Minor refactor.
- Non-blocking visual polish.

## Current Structured Markers

Status: closed
Severity: P0
ID: BUG-043-001
Title: Raw invite leakage gate covered by regression tests and redaction script.

Status: closed
Severity: P1
ID: BUG-043-009
Title: Release blocker script fails when open P0/P1 items are present.
