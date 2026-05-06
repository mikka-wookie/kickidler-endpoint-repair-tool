# GUI MVP Spec

## Purpose

The GUI is a support dashboard for technical support engineers. It exposes safe workflow actions without requiring CLI syntax knowledge.

## Layout

1. Header
   - App name and version.
   - Active profile.
   - Administrator status.
   - Latest RunID.

2. Quick Actions
   - Check.
   - Verify.
   - Collect Bundle.
   - Open Reports Folder.

3. Repair Preparation
   - Installer path.
   - Browse button.
   - Masked invite input.
   - Profile field.
   - Preflight.
   - Repair Dry-Run.
   - Visually separated `RUN REAL REPAIR`.

4. Current Result
   - Workflow status.
   - Health status.
   - Install mode.
   - Primary issue code.
   - Recommendation.
   - Report directory.

5. Timeline
   - Operation ID.
   - Status.
   - Short message.
   - Duration.
   - Failure category when failed.

6. Details
   - Summary path.
   - Operations path.
   - Primary result path.
   - Support bundle path.

## Behavior Rules

- Only one workflow may run at a time.
- Actions are disabled while a workflow is running.
- Cancellation uses workflow context cancellation.
- Invite is masked in the input control.
- Invite is cleared after workflow completion or cancellation.
- Invite is never serialized in GUI request JSON.
- GUI calls the workflow service directly and does not launch `kigrepair.exe`.
- Real repair requires administrator rights and exact confirmation text `YES`.
- Errors shown by default are redacted and support-readable.
- Raw stack traces are not shown by default.

## Safety Rules

- Read-only actions remain read-only.
- GUI does not bypass backend safety gates.
- GUI does not perform direct cleanup, service, process, registry, Defender, MSI, or file mutation.
- Backend remains responsible for validation, rollback, operation logging, reports, and exit classification.
