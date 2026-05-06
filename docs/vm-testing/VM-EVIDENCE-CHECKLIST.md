# VM Evidence Checklist

Collect evidence per scenario under:

```text
evidence\<scenario-id>\
```

Required files:

- `evidence-manifest.json`
- command stdout/stderr under `command-output\`
- copied latest `C:\ProgramData\kigrepair\Reports\<timestamp>\` when relevant
- `version --json` output
- release manifest if available
- checksums if available
- Windows version
- Defender status summary
- admin/elevation status
- list of generated report directories
- support bundle path if created
- screenshots index at `screenshots\README.txt`

Do not collect:

- Raw invite values.
- Unrelated customer files.
- Full user profiles.
- Browser history.
- Arbitrary `ProgramData` contents.
- Installer MSI files unless release owner explicitly approves that evidence handling.

Run evidence collection:

```powershell
.\scripts\vm-tests\collect-vm-evidence.ps1 `
  -ScenarioId "VM-001" `
  -OutDir ".\evidence\VM-001" `
  -KigrepairPath ".\kigrepair.exe"
```

Run redaction check before attaching evidence:

```powershell
.\scripts\vm-tests\run-redaction-check.ps1 `
  -Path ".\evidence\VM-001" `
  -SecretPattern "REAL-SECRET-INVITE"
```

