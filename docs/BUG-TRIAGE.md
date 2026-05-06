# Step 40 Bug Triage

This triage is the stabilization gate for the simple support-dashboard MVP.

## P0 - safety blocker

- Invite leak in GUI/service responses: fixed in Step 40 by per-run sensitive value redaction in the workflow service response path.
- Destructive action without admin/confirmation/rollback: no regression found in this pass. Real repair still requires administrator rights, exact GUI confirmation text `YES`, backend `Yes`, installer validation, and rollback snapshot before mutation.
- Wrong cleanup target or path-trust bypass: no code changes made to cleanup target validation. Existing cleanup safety remains the release gate.
- Hidden WMI false positive or process/service mutation on path mismatch: no mutation logic was changed. Existing service/process trust tests remain required release gates.

## P1 - workflow blocker

- GUI result contract was too thin for support use: fixed by extracting health, install mode, primary issue, recommendation, report paths, support bundle path, and timeline fields from workflow responses.
- Workflow service could return unredacted error/result text when a GUI request carried an invite: fixed by redacting errors, warnings, timeline, and result payloads against request-sensitive values.
- Collect-report bundle reference was inferred even for non-collect workflows: fixed so support bundle path is taken from the workflow result and only inferred for collect-report.
- Report files on workflow failure: covered by a workflow-service regression test for failed workflows after report initialization.

## P2 - support usability

- Main GUI was too command-list oriented: simplified into one support dashboard with header, quick actions, repair preparation, current result, timeline, and details.
- Timeline lacked duration/failure category: fixed in GUI view model formatting.
- Real repair action needed clearer separation: made the button text visually explicit as `RUN REAL REPAIR` and kept exact `YES` confirmation.
- Report path visibility: current result and details panel always show latest report references when a workflow returns them.

## P3 - cosmetic/internal

- Minor view-model duplication remains acceptable for MVP.
- Native Win32 UI is intentionally simple. A richer GUI framework is deferred until workflow stability is proven.

## Release Gate

P0 items must remain fixed by tests and manual redaction checks. P1 items are MVP blockers if they recur during validation.
