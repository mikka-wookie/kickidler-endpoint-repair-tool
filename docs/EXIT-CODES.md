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
| `4` | Invalid input. | Check installer path, invite presence, flags, and command syntax. |
| `5` | Cleanup failed. | Review `cleanup-result.json` and `operations.json`. |
| `6` | Install failed. | Review `install-result.json` and `msi-install.log`. |
| `7` | Validation failure, not ready, failed verification, or safety validation failure. | Review blocker details before mutation. |
| `8` | Defender ensure failed. | Check Defender policy, Tamper Protection, and `defender-result.json`. |
| `9` | Reboot required. | Schedule reboot if support policy allows. |
| `10` | Unexpected error. | Collect support bundle and escalate. |

## Common Automation Mapping

For high-level automation, treat `0` as pass, `1` as pass with warnings, `7` as validation or verification failure, and `10` as unexpected error. Other codes give more specific failure categories.

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
- `5` cleanup failed.
- `6` install failed.
- `7` final verification failed.
- `8` Defender ensure failed.
- `10` unexpected error.
