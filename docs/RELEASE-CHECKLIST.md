# kigrepair Release Checklist

Use this checklist before distributing a new internal or support release.

## A. Before Build

- Confirm the repository is clean or all local changes are intentional.
- Run or plan to run the standard validation commands, including `go test ./...`.
- Select the release version.
- Update release notes when applicable.
- Search docs, config, and examples for credentials, tokens, raw invite values, and customer data.
- Confirm an approved signing certificate is available if this is a signed support release.

## B. Build

Unsigned internal build:

```powershell
.\scripts\build-release.ps1 -Version "0.1.0-dev" -Clean
```

Signed support build:

```powershell
.\scripts\build-release.ps1 -Version "0.1.0" -IncludeGui -Sign -CertificateThumbprint "<THUMBPRINT>" -TimestampUrl "<TIMESTAMP_URL>" -Clean
```

Capture script output in the internal release ticket.

## C. Validate

```powershell
.\scripts\validate-release.ps1 -ReleaseDir ".\dist\kigrepair-0.1.0-windows-amd64" -ZipPath ".\dist\kigrepair-0.1.0-windows-amd64.zip"
```

For signed releases:

```powershell
.\scripts\validate-release.ps1 -ReleaseDir ".\dist\kigrepair-0.1.0-windows-amd64" -ZipPath ".\dist\kigrepair-0.1.0-windows-amd64.zip" -RequireSigned
```

Verify:

- `checksums.txt` matches release folder files.
- `RELEASE-MANIFEST.json` parses and lists `kigrepair.exe`.
- `SIGNATURES.txt` records unsigned warning or successful signing.
- `kigrepair.exe version --json` shows version, commit, build date, built by, Go version, OS/arch, signed status, and executable path.

## D. Smoke Test

```powershell
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe version
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe check
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe verify
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe config show
.\dist\kigrepair-0.1.0-windows-amd64\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Run a redaction search with a known placeholder from your test case:

```powershell
Get-ChildItem ".\dist\kigrepair-0.1.0-windows-amd64" -Recurse -File | Select-String -Pattern "<REDACTED>"
```

Expected: no matches.

Check for bundled MSI files:

```powershell
Get-ChildItem ".\dist\kigrepair-0.1.0-windows-amd64" -Recurse -Filter *.msi
```

Expected: no results.

## E. Publish

Upload or attach:

- `kigrepair-<version>-windows-amd64.zip`
- `kigrepair-<version>-windows-amd64.zip.sha256`
- `RELEASE-MANIFEST.json`
- `SIGNATURES.txt`
- release notes or ticket link

Record whether the build is signed or unsigned.

## F. Rollback

- Keep the previous release zip and checksum.
- Document how support should revert to the prior release.
- Do not delete previous release artifacts until the new release has passed support smoke testing.

## Never Include

- Raw invite values.
- Customer logs unless they are part of an approved redacted support bundle.
- Grabber MSI files unless explicitly approved for that distribution.
- Certificate private keys or export files.
