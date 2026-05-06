# Release Trust

`kigrepair` release artifacts are built to be traceable and verifiable by support engineers and internal testers.

## Release Folder Contents

Expected folder:

```text
dist\kigrepair-<version>-windows-amd64\
```

Expected files:

- `kigrepair.exe`
- `kigrepair-gui.exe` when built with `-IncludeGui`
- `README.md`
- `SUPPORT-RUNBOOK.md`
- `docs\`
- `assets\README.txt`
- `examples\commands.ps1`
- `checksums.txt`
- `RELEASE-MANIFEST.json`
- `SIGNATURES.txt`

The release zip is created next to the folder:

```text
dist\kigrepair-<version>-windows-amd64.zip
dist\kigrepair-<version>-windows-amd64.zip.sha256
```

## Verify Checksums

Validate the whole release:

```powershell
.\scripts\validate-release.ps1 -ReleaseDir ".\dist\kigrepair-<version>-windows-amd64" -ZipPath ".\dist\kigrepair-<version>-windows-amd64.zip"
```

Manual file hash check:

```powershell
Get-FileHash ".\dist\kigrepair-<version>-windows-amd64\kigrepair.exe" -Algorithm SHA256
Get-Content ".\dist\kigrepair-<version>-windows-amd64\checksums.txt"
```

## Verify Authenticode Signature

Signed releases should pass:

```powershell
Get-AuthenticodeSignature ".\dist\kigrepair-<version>-windows-amd64\kigrepair.exe"
.\scripts\validate-release.ps1 -ReleaseDir ".\dist\kigrepair-<version>-windows-amd64" -RequireSigned
```

`SIGNATURES.txt` records signing and verification command results. Unsigned internal builds are allowed, but `SIGNATURES.txt` must clearly state that signing was not requested.

## Version Check

```powershell
.\dist\kigrepair-<version>-windows-amd64\kigrepair.exe version
.\dist\kigrepair-<version>-windows-amd64\kigrepair.exe version --json
```

Confirm version, commit, build date, built by, Go version, OS/arch, signed status, and executable path.

## MSI Policy

The Grabber MSI is not bundled by default. This prevents accidental distribution of the wrong installer and keeps release artifacts separate from customer-specific deployment instructions.

If support instructs you to use an MSI, place the approved file under `assets\` locally. Do not place invite values or credentials in `assets\`.

## Distribution Rules

Support engineers should distribute the release zip plus `.sha256` sidecar through the approved internal channel. Attach `RELEASE-MANIFEST.json` and `SIGNATURES.txt` to the internal release ticket.

Do not include:

- Raw invite values.
- Customer logs unless they are part of an approved redacted support bundle.
- Grabber MSI files unless explicitly approved.
- Certificate private keys.
