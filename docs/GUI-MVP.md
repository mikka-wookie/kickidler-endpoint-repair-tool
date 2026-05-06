# kigrepair GUI MVP

`kigrepair-gui.exe` is an internal Windows support-engineer shell over the same workflow service used by the CLI. It is not a separate repair implementation, does not shell out to `kigrepair.exe`, and does not parse console output.

Build:

```powershell
go build -o kigrepair-gui.exe ./cmd/kigrepair-gui
```

Run:

```powershell
.\kigrepair-gui.exe
```

The GUI writes reports to the normal report location:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

## Available Actions

Read-only actions:

- Check
- Verify
- Preflight
- Repair Dry-Run
- Collect Report
- Reports List
- Reports Cleanup Dry-Run

Mutating action:

- Real Repair

Real Repair requires administrator rights, installer path, invite value, workflow policy/preflight readiness, rollback snapshot support from the workflow, and an explicit destructive confirmation. The confirmation prompt requires typing exact `YES`.

## Invite Handling

The invite field is masked. The GUI passes the invite only in the typed workflow request and clears the input after workflow completion or cancellation where practical.

The raw invite must not appear in the window title, timeline, status text, error text, report files, logs, support bundle, config, or request JSON. GUI rendering passes through redaction helpers, and workflow request structs do not serialize `InviteValue`.

## Admin Handling

The dashboard shows whether the GUI is elevated. Real Repair is blocked before workflow execution when the GUI is not elevated. The `Restart as Administrator` button uses the existing Windows elevation wrapper and never auto-elevates without user action.

## Limitations

This is an MVP support tool. Reports remain the source of evidence, and CLI commands remain supported for automation, escalation, and exact reproducibility.
