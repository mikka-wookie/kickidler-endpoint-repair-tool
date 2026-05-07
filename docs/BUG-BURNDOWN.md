# Bug Burndown

This file is the Step 43 MVP bug tracker. P0 and P1 items block an internal MVP candidate until closed or explicitly moved to a lower severity with evidence.

| ID | Severity | Area | Title | Status | Repro steps | Expected | Actual | Owner | Fixed in | Regression test |
|---|---|---|---|---|---|---|---|---|---|---|
| BUG-043-001 | P0 | Security | Raw invite must never appear in workflow output, GUI state, reports, or support bundles | closed | Run preflight/repair dry-run with a fake invite and inspect JSON, GUI state, reports, and bundle metadata | Invite is redacted or omitted everywhere after request entry | Previously observed risk during backend-to-GUI contract hardening | engineering | Step 43 baseline | `internal/ui/state_test.go`, `internal/app/workflowservice/service_test.go`, `scripts/check-redaction.ps1` |
| BUG-043-002 | P1 | GUI | Startup must not freeze or run backend workflows | closed | Launch GUI or construct startup view | Empty visible state appears without workflow service calls or report scanning | Previously observed blank/frozen startup risk | engineering | Step 43 baseline | `TestStartupViewDoesNotCallWorkflowService` |
| BUG-043-003 | P1 | GUI | Long workflows must not block normal UI state transitions | closed | Start Check, Verify, Collect Bundle, Preflight, and Repair Dry-Run | Running state is visible immediately, action buttons disable, cancel is wired, buttons re-enable on completion | Previously observed Not Responding risk | engineering | Step 43 baseline | `TestGUIWorkflowActionsUseControllerRunningState`, `TestOnlyOneWorkflowCanRunAtATimeAndCancelUsesContext`, `TestButtonStateWhileRunning` |
| BUG-043-004 | P1 | GUI/Backend contract | Not-ready repair dry-run must preserve blocking reasons and support summaries | closed | Repair dry-run with missing admin and missing installer | Status is `not_ready`; admin and installer blockers, primary issue, recommendation, and report dir are visible | Previously observed mapping loss | engineering | Step 43 baseline | `TestMapNotReadyRepairPlanToGUIState`, `TestRepairDryRunOutcomeContractAndRedaction` |
| BUG-043-005 | P1 | GUI | Repeated unsafe process warnings must be compact in the GUI timeline | closed | Generate 90+ skipped unsafe process warnings | GUI shows aggregate entry; full detail remains in JSON reports | Timeline flood could hide the primary issue | engineering | Step 43 baseline | `TestWarningAggregationCollapsesProcessFlood`, `TestOutcomeAggregatesProcessWarningsAndCapsTimeline` |
| BUG-043-006 | P1 | Release | GUI binary must launch without a console window | closed | Inspect build script or build release with GUI | GUI uses `-H windowsgui`; CLI build does not | Previously observed console window risk | engineering | Step 43 baseline | `TestReleaseScriptBuildsGUIWithoutConsoleWindow`, `TestBuildScriptKeepsWindowsGuiFlagGUIOnly` |
| BUG-043-007 | P1 | GUI | Report file buttons must fail safely on missing paths | closed | Click Open Report Folder/Open Summary/Open Operations/Copy Report Path before a run or after deleting report files | UI handles empty or missing paths without panic | Missing report files could produce unclear errors | engineering | Step 43 baseline | `TestOpenPathMissingFileReturnsSafeError`, `TestFileButtonEmptyStateIsSafe` |
| BUG-043-008 | P0 | Repair safety | Real repair must require exact `YES` and backend safety gates | closed | Attempt real repair with `yes`, `Yes`, missing installer, missing invite, non-admin | GUI rejects non-exact confirmation and backend blocks unsafe repair | Real repair could otherwise mutate systems | engineering | Step 43 baseline | `TestRealRepairRequiresExactYES`, `TestRealRepairConfirmationRejectsCancelAndWhitespace`, `internal/repair/*_test.go` |
| BUG-043-009 | P1 | Release gate | Open P0/P1 blockers must be impossible to ignore | closed | Add open P0/P1 to bug tracker or blocker JSON and run blocker script | Script exits non-zero; readiness is NO-GO | Manual checklist could miss blockers | engineering | Step 43 | `internal/release/mvp_gate_test.go` |

## Severity Definitions

- P0 - safety blocker.
- P1 - MVP blocker.
- P2 - support usability issue.
- P3 - cosmetic or internal cleanup.

## Operating Rule

Do not move a P0/P1 issue to limitations. P0/P1 issues remain blockers until fixed, verified, or explicitly waived by release ownership outside this repository.
