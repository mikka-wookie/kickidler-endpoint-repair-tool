# kigrepair v0.1.0

## Summary

Windows support utility for Kickidler Grabber diagnostics, cleanup, install, Defender exclusion handling, repair orchestration, and support bundle collection.

## Included commands

- check
- cleanup --dry-run
- cleanup
- install
- defender
- repair
- collect-report
- version

## Safety notes

- Detection and collect-report are read-only.
- cleanup --dry-run is read-only.
- cleanup, install, defender --ensure, and repair require administrator rights.
- Interactive mode can request UAC elevation.
- quiet/non-interactive mode does not trigger UAC and requires elevated shell.
- Raw invite values are not logged.

## MVP limitations

- Some service/process operations may use sc.exe, taskkill.exe, and PowerShell.
- Native Windows APIs may replace these later.
- GUI is not implemented yet.
- Code signing is not implemented by this script unless added later.

## Validation

- Build date:
- Commit:
- Tested on:
- Notes:
