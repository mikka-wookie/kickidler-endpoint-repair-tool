# Regression Test Plan

This plan maps previously observed Step 43 failures to automated tests, smoke scripts, or manual GUI checklist items.

| Risk | Coverage |
|---|---|
| GUI startup freeze | `TestStartupViewDoesNotCallWorkflowService`; GUI smoke checklist confirms visible startup within 1 second |
| GUI workflow freeze | `TestGUIWorkflowActionsUseControllerRunningState`, `TestOnlyOneWorkflowCanRunAtATimeAndCancelUsesContext`, `TestButtonStateWhileRunning`; GUI smoke checklist |
| Console window on GUI launch | `TestReleaseScriptBuildsGUIWithoutConsoleWindow`, `TestBuildScriptKeepsWindowsGuiFlagGUIOnly` |
| Not-ready repair dry-run mapping | `TestMapNotReadyRepairPlanToGUIState`, `TestRepairDryRunOutcomeContractAndRedaction` |
| Warning flood aggregation | `TestWarningAggregationCollapsesProcessFlood`, `TestOutcomeAggregatesProcessWarningsAndCapsTimeline` |
| Invite redaction | `TestResultRenderingDoesNotExposeInvite`, `TestInviteNotStoredInGUIStateOrTimeline`, `TestExecuteWorkflowProgressAndRedaction`, `scripts/check-redaction.ps1` |
| File buttons | `TestOpenPathMissingFileReturnsSafeError`, `TestFileButtonEmptyStateIsSafe`, GUI smoke checklist |
| Real repair gate | `TestRealRepairRequiresExactYES`, `TestRealRepairConfirmationRejectsCancelAndWhitespace`, `TestRealRepairAcceptedSetsYesAndRedactsInvite`, backend repair safety tests |
| Release blocker enforcement | `internal/release/mvp_gate_test.go`, `scripts/check-release-blockers.ps1` |
| MVP readiness decision | `internal/release/mvp_gate_test.go`, `scripts/generate-mvp-readiness.ps1` |
| Smoke result model | `internal/release/mvp_gate_test.go`, `scripts/smoke-mvp.ps1` |
| Redaction model | `internal/release/mvp_gate_test.go`, `scripts/check-redaction.ps1` |

## Manual Validation Required

Automated tests do not prove Windows UI responsiveness or console-window behavior on every endpoint. Run [GUI Smoke Checklist](GUI-SMOKE-CHECKLIST.md) on a disposable Windows VM before marking an MVP candidate.
