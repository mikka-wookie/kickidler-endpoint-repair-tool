# kigrepair Report Files

Reports are written under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

The support bundle is usually:

```text
kigrepair-support-bundle.zip
```

## Command Safety Classes

Read-only / non-destructive except report writing: `check`, `verify`, `preflight`, `repair --dry-run`, `cleanup --dry-run`, `collect-report`, `reports list`, `reports cleanup --dry-run`, `version`.

System-modifying / destructive: `cleanup --yes`, `repair --yes`, `install --yes`, `defender ensure --yes`, `reports cleanup --yes`.

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## Redaction

Invite values are treated as sensitive and should be redacted from console output, logs, JSON reports, and support bundles. Token, access token, refresh token, password, secret, and Authorization Bearer/Basic values are also redacted. If a support engineer sees a raw invite in any report, do not attach that file until it is reviewed and sanitized.

## Run ID and Timeline

Each workflow run gets a `Run ID` such as `kigrun-20260506-120000-a1b2c3`. Use it to link `summary.txt`, `operations.json`, `repair.log`, and result files from the same execution.

`summary.txt` includes a compact workflow timeline with operation IDs, status, duration, and failure category when available. Start with the first `failed` or `warning` entry.

`operations.json` is the structured timeline. Operation IDs use the format `op-001-detect`, `op-002-classify`, and so on.

`repair.log` is JSON Lines. Each line is a redacted structured event with timestamp, level, run ID, workflow, optional operation ID, event name, status, category, duration, and extra fields.

Policy metadata appears in `config-metadata.json` and compactly in policy-aware results such as `preflight-result.json`, `repair-plan.json`, `repair-result.json`, `cleanup-plan.json`, `cleanup-result.json`, `defender-result.json`, `operations.json`, and `summary.txt`. Use it to confirm the active profile, CLI profile override, and safety gates such as rollback, Defender coverage, hidden WMI cleanup, and detection-unknown blocking.

Workflow service responses return paths to report files instead of embedding support evidence blobs. Reports remain the source of evidence for CLI and any future GUI.

The GUI MVP displays report paths returned by the workflow service. It does not create a separate evidence format; `summary.txt`, `operations.json`, primary result JSON, `repair.log`, and `kigrepair-support-bundle.zip` remain the support artifacts.

## File Reference

| File | Purpose | Produced by | How support should read it | Warnings/errors | Included in bundle |
| --- | --- | --- | --- | --- | --- |
| `initial-detection.json` | Starting service, file, registry, process, Defender, and health state. | `check`, `verify`, `cleanup`, `repair` | Start here for original endpoint state. | Yes | Yes |
| `final-detection.json` | State after a mutating workflow or verification stage. | `cleanup --yes`, `install --yes`, `repair --yes` | Compare with initial detection. | Yes | Yes |
| `operations.json` | Structured operation results with step, target, status, message, error, timestamp. | Most workflows | Review failed and warning entries first. | Yes | Yes |
| `summary.txt` | Human-readable workflow summary. | Most workflows | Use for quick ticket notes, excluding invites. | Yes | Yes |
| `repair.log` | Operational log. | Most workflows | Use for chronological troubleshooting. | Yes | Yes |
| `cleanup-plan.json` | Validated cleanup targets and skipped targets. | `cleanup --dry-run`, `cleanup --yes`, `repair` | Confirm every deletion/service/process target is expected. | Yes | Yes |
| `cleanup-result.json` | Actual cleanup execution result. | `cleanup --yes`, `repair --yes` | Review success, skipped, failed cleanup steps. | Yes | Yes |
| `install-result.json` | MSI install result and verification data. | `install --yes`, `repair --yes` | Check MSI exit code and final install status. | Yes | Yes |
| `defender-result.json` | Defender status or ensure result. | `defender`, `defender --ensure`, `repair` | Check status availability, exclusions, and policy/tamper warnings. | Yes | Yes |
| `repair-result.json` | High-level repair outcome. | `repair --yes` | Use as the repair decision summary. | Yes | Yes |
| `collect-result.json` | Diagnostic collection result and bundle manifest. | `collect-report` | Confirm bundle path and any skipped collectors. | Yes | Yes |
| `verification-result.json` | Final or standalone verification result. | `verify`, `repair --yes`, `install --yes` | Review failed hard requirements and warnings. | Yes | Yes |
| `classification-result.json` | Support-facing primary issue and secondary issues. | `check`, `verify`, `repair` | Use primary issue for escalation routing. | Yes | Yes |
| `recommendation-result.json` | Next-action recommendation. | `check`, `verify`, `repair` | Use primary action before choosing repair or escalation. | Yes | Yes |
| `preflight-result.json` | Readiness checks for repair. | `preflight`, `repair --dry-run`, `repair --yes` | Confirm blockers before mutation. | Yes | Yes |
| `repair-plan.json` | Ordered repair execution plan. | `repair --dry-run`, `repair --yes` | Review before real repair. | Yes | Yes |
| `installer-validation.json` | MSI path, metadata, and validation result. | `preflight`, `install`, `repair` | Verify source, name, and hash where available. | Yes | Yes |
| `rollback-info.json` | Snapshot metadata created before mutation. | `cleanup --yes`, `repair --yes` | Use for audit and escalation context. | Yes | Yes |
| `report-cleanup-plan.json` | Planned old report directories to remove. | `reports cleanup --dry-run`, `reports cleanup --yes` | Confirm all paths stay under report root. | Yes | Yes |
| `report-cleanup-result.json` | Actual report cleanup result. | `reports cleanup --yes` | Review deleted/skipped report folders. | Yes | Yes |
| `msi-install.log` | Verbose MSI install log. | `install --yes`, `repair --yes` | Search for `Return value 3` and MSI error codes. | Yes | Yes |
| `msi-uninstall.log` | Verbose MSI uninstall log. | `cleanup --yes`, `repair --yes` | Search for uninstall failure codes. | Yes | Yes |
| `kigrepair-support-bundle.zip` | Escalation archive containing reports and diagnostics. | `collect-report` | Attach to escalation ticket after invite redaction check. | Yes | No, it is the bundle |

## Workflow File Map

- `check`: `initial-detection.json`, `classification-result.json`, `recommendation-result.json`, `operations.json`, `summary.txt`, `repair.log`.
- `verify`: `initial-detection.json`, `verification-result.json`, `classification-result.json`, `recommendation-result.json`, `operations.json`, `summary.txt`, `repair.log`.
- `preflight`: `preflight-result.json`, `installer-validation.json`, `operations.json`, `summary.txt`, `repair.log`.
- `cleanup --dry-run`: `initial-detection.json`, `cleanup-plan.json`, `operations.json`, `summary.txt`, `repair.log`.
- `cleanup --yes`: cleanup dry-run files plus `cleanup-result.json`, `final-detection.json`, `rollback-info.json`, and possibly `msi-uninstall.log`.
- `install --yes`: `installer-validation.json`, `install-result.json`, `final-detection.json`, `verification-result.json`, `msi-install.log`, `operations.json`, `summary.txt`, `repair.log`.
- `repair --dry-run`: `repair-plan.json`, `preflight-result.json`, `initial-detection.json`, `cleanup-plan.json`, `classification-result.json`, `recommendation-result.json`, `installer-validation.json`, `operations.json`, `summary.txt`, `repair.log`.
- `repair --yes`: dry-run files plus execution results, final detection, verification, rollback, and MSI logs.
- `collect-report`: `collect-result.json`, `summary.txt`, `repair.log`, `kigrepair-support-bundle.zip`.
- `reports cleanup`: `report-cleanup-plan.json`, and for real cleanup `report-cleanup-result.json`.
