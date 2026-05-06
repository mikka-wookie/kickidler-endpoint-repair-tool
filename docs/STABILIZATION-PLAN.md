# Step 40 Stabilization Plan

## Scope

Stabilize backend workflow contracts and reset the GUI into a simple support-friendly MVP. Do not add broad new features or weaken safety gates.

## Backend Focus

1. Preserve existing CLI behavior and exit-code mapping.
2. Keep destructive workflows gated by administrator checks, `--yes`/confirmation, installer validation, rollback snapshot, Defender policy, and cleanup safety validation.
3. Redact invite values in workflow responses, errors, timeline, reports, and GUI-visible text.
4. Keep read-only workflows read-only: check, verify, preflight, repair dry-run, collect-report, and reports list.
5. Ensure failed workflows still return report references when report initialization succeeded.

## GUI Focus

1. One workflow at a time.
2. Disable actions while a workflow runs.
3. Mask invite input and clear it after completion or cancellation.
4. Do not store invite and do not serialize it.
5. Do not shell out to `kigrepair.exe`; call `internal/app/workflowservice`.
6. Show support-readable status, health, install mode, primary issue, recommendation, report directory, and timeline.
7. Keep real repair visually distinct and require exact `YES`.

## Validation

Run:

```powershell
gofmt -w .
go mod tidy
go test ./...
go build -o kigrepair.exe ./cmd/kigrepair
go build -o kigrepair-gui.exe ./cmd/kigrepair-gui
```

Manual smoke:

```powershell
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe collect-report
.\kigrepair-gui.exe
```

Redaction check:

```powershell
Select-String -Path "C:\ProgramData\kigrepair\Reports\<timestamp>\*" -Pattern "REAL-SECRET-INVITE" -Recurse
```

Expected: no matches.

## Deferred

- Rich GUI styling framework.
- New workflows.
- Broad architecture changes.
