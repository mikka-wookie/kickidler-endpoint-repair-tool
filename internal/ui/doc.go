// Package ui is reserved for a future Windows GUI.
//
// A GUI must call internal/app.WorkflowService through an adapter such as
// internal/app/workflowservice. It must not shell out to kigrepair.exe, parse
// console text, or bypass policy, safety, confirmation, admin, rollback, or
// report-writing gates.
//
// GUI input controls may temporarily hold an invite in memory, but raw invite
// values must not be logged, serialized, displayed outside the input control,
// written to reports, or included in support bundles.
package ui
