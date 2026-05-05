# kigrepair Exit Codes

Exit codes are designed for support scripts, RMM tools, and manual triage. Always inspect report files when the exit code is not `0`.

## Command Safety Classes

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Global Exit Code Table

| Code | Meaning | Support action |
| --- | --- | --- |
| `0` | Success, passed, planned, or ready. | Continue normal flow. |
| `1` | Warning, partial success, or ready with warnings. | Review warnings in `summary.txt` and JSON result files. |
| `2` | Administrator rights required. | Re-run from elevated PowerShell or approved RMM context. |
| `3` | Confirmation declined or missing. | Add `--yes` only when real mutation is approved. |
| `4` | Reserved legacy invalid-input code. | Current hardened workflows should return `7` instead. |
| `5` | Reserved legacy cleanup-failed code. | Current hardened workflows should return `7` instead. |
| `6` | Reserved legacy install-failed code. | Current hardened workflows should return `7` instead. |
| `7` | Validation failure, not ready, failed verification, or safety validation failure. | Review blocker details before mutation. |
| `8` | Reserved legacy Defender-failed code. | Current hardened workflows should return `7` instead. |
| `9` | Reboot required. | Schedule reboot if support policy allows. |
| `10` | Unexpected error. | Collect support bundle and escalate. |

## Common Automation Mapping

For high-level automation, treat `0` as pass, `1` as pass with warnings, `7` as validation, expected operational, or verification failure, and `10` as unexpected error.

## Failure-Mode Policy

- Report directory creation failure exits `10` before mutation.
- Missing invite, missing installer, invalid installer, failed preflight, failed verification, MSI failure, Defender ensure failure, and cleanup safety blockers are expected operational failures and map to `7`.
- Partial diagnostics, unavailable optional collectors, Defender query unavailable in read-only checks, and bundle warnings should map to `1`.
- Panics and unhandled infrastructure failures map to `10`. Stack traces are hidden unless `KIGREPAIR_DEBUG=1`.
- Command output and report files must not include raw invite values, bearer tokens, passwords, or secrets.

## Command-Specific Notes

### verify

- `0` passed.
- `1` warning.
- `7` failed verification.
- `10` unexpected error.

### preflight

- `0` ready.
- `1` ready with warnings.
- `7` not ready.
- `10` unexpected error.

### repair --dry-run

- `0` planned.
- `1` planned with warnings.
- `7` not ready.
- `10` unexpected error.

### reports cleanup

- `0` success.
- `1` partial success or warnings.
- `7` missing `--yes` for real cleanup or safety validation failure.
- `10` unexpected error.

### repair --yes

- `0` success.
- `1` completed with warnings.
- `2` administrator rights required.
- `3` confirmation missing.
- `7` validation, cleanup, install, Defender, or final verification failed.
- `10` unexpected error.
