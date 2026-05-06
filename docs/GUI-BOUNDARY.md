# GUI Boundary

`kigrepair` remains CLI-first. No GUI is implemented yet.

Future GUI code must call `internal/app.WorkflowService`, using an implementation such as `internal/app/workflowservice`. The GUI must not shell out to `kigrepair.exe`, parse console output, or duplicate workflow logic.

The workflow service boundary provides:

- typed request/response structs for check, verify, preflight, repair plan, repair, cleanup plan, cleanup, collect-report, reports, and config workflows
- consistent response metadata: run ID, workflow, status, exit code, report directory, summary path, operations path, primary result path, and timing
- progress events for workflow and operation start/finish
- confirmation interfaces for future GUI modal prompts
- context cancellation at the service boundary
- a machine-readable workflow catalog

Read-only workflows:

- `check`
- `verify`
- `preflight`
- `repair_dry_run`
- `cleanup_dry_run`
- `collect_report`
- `reports_list`
- `reports_cleanup_dry_run`
- `config_show`
- `config_validate`

Mutating workflows:

- `repair`
- `cleanup`
- `reports_cleanup`

Mutating workflows must continue to enforce existing admin, `--yes`, policy, safety, installer validation, and rollback gates inside the shared workflow layer.

Invite handling:

- request structs may hold `InviteValue` in memory
- `InviteValue` must use `json:"-"`
- raw invite must never be logged, serialized, printed, written to reports, or included in support bundles
- UI may display raw invite only inside the active input control

Report files remain the support evidence source. GUI responses should reference report files rather than embedding large report blobs.

