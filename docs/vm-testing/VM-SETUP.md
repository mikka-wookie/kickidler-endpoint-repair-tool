# VM Setup

## Required Tools

- Windows 10 or Windows 11 VM.
- PowerShell 5.1 or newer.
- Admin account.
- Non-admin test account.
- Approved Grabber MSI copied manually into `.\assets\`.
- `kigrepair` release zip or locally built binaries.
- Optional Sysinternals tools only if approved for the test environment.
- VM snapshot support.

Do not commit real installer files or invite values.

## Directory Layout

Create this local layout on the VM:

```text
C:\kigrepair-test\
  kigrepair.exe
  kigrepair-gui.exe
  assets\
    README.txt
    grabberEM.x64.msi
  evidence\
  scripts\
```

`kigrepair-gui.exe` and the MSI are optional for read-only CLI smoke. The MSI must be manually provided from an approved source.

## Snapshot Names

Use stable snapshot names so results can be compared across releases:

- `clean-baseline`
- `healthy-standard-install`
- `broken-service-binary`
- `partial-msi-leftovers`
- `hidden-wmi-baseline`
- `post-repair-validation`

## Invite Handling

Use an environment variable for local test sessions:

```powershell
$env:KIGREPAIR_TEST_INVITE = "<INVITE>"
```

Never store invite values in scripts, config, documentation, tickets, screenshots, release artifacts, reports, or evidence bundles. Use `<INVITE>` or `<REDACTED>` in written records.

## Safety

- Take a snapshot before destructive tests.
- Run destructive tests only on disposable VM snapshots.
- Never run destructive VM tests on production or customer machines.
- Never paste real invite values into docs, tickets, screenshots, or support bundles.
- Do not store invite values in scripts.
- Use PowerShell prompt input or environment variables only for local operator input.
- Do not manually modify normal Windows `svchost.exe`, `WmiPrvSE.exe`, or `RuntimeBroker.exe`.
- Hidden WMI tests may touch only the exact approved hidden WMI test path and only when that scenario is explicitly selected.

