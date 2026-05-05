# kigrepair Command Reference

All workflow commands except `version` write reports under:

```text
C:\ProgramData\kigrepair\Reports\<timestamp>\
```

Use `--config "C:\Path\kigrepair.yaml"` to load optional support defaults. CLI flags override config values. See [Configuration](CONFIGURATION.md).

Failure handling: read-only commands should continue with warnings when service, process, registry, or Defender queries are partially unavailable. Mutating commands should stop before mutation when critical prerequisites fail, including report directory creation, required rollback snapshot writing, missing invite, missing installer, invalid installer, missing admin rights, or missing confirmation. External command details are redacted in logs and reports.

## Command Safety Classes

Read-only / non-destructive except report writing:

- `check`
- `verify`
- `preflight`
- `repair --dry-run`
- `cleanup --dry-run`
- `collect-report`
- `reports list`
- `reports cleanup --dry-run`
- `version`

System-modifying / destructive:

- `cleanup --yes`
- `repair --yes`
- `install --yes`
- `defender ensure --yes` using `.\kigrepair.exe defender --ensure --yes`
- `reports cleanup --yes`

Do not paste real invite values into tickets, screenshots, or shared logs. `kigrepair` output should redact invite values. Command examples must use `<INVITE>`. Support bundles should not contain raw invite values.

## version

Purpose: Prints build metadata.

Destructive: No.

Admin required: No.

Required inputs: None.

Writes: No report files.

Example:

```powershell
.\kigrepair.exe version
.\kigrepair.exe version --json
```

Expected exit codes: `0` success, `10` unexpected error.

## config

Purpose: Prints, validates, or samples optional YAML configuration.

Destructive: No.

Admin required: No.

Examples:

```powershell
.\kigrepair.exe config sample
.\kigrepair.exe config validate --config ".\kigrepair.yaml"
.\kigrepair.exe config show --config ".\kigrepair.yaml" --json
```

Expected exit codes: `0` success, `4` invalid config, `10` unexpected error.

## check

Purpose: Detects current Grabber installation state and health.

Destructive: No, except report writing.

Admin required: No, but some Windows data may be unavailable without elevation.

Required inputs: None.

Writes: `initial-detection.json`, `classification-result.json`, `recommendation-result.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe check
```

Expected exit codes: `0` healthy, `1` warning or not installed depending workflow result, `7` failed validation, `10` unexpected error.

## verify

Purpose: Verifies current Grabber service, executable, install root, and warning conditions without changing the system.

Destructive: No, except report writing.

Admin required: No, but elevated access can improve detection.

Required inputs: None.

Writes: `verification-result.json`, `initial-detection.json`, `classification-result.json`, `recommendation-result.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe verify
```

Expected exit codes: `0` passed, `1` warning, `7` failed verification, `10` unexpected error.

## preflight

Purpose: Checks whether repair inputs and system prerequisites are ready without modifying the system.

Destructive: No, except report writing.

Admin required: No, but missing admin is reported as a blocker for real repair.

Required inputs: `--installer`, `--invite`.

Writes: `preflight-result.json`, `installer-validation.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Expected exit codes: `0` ready, `1` ready with warnings, `7` not ready, `10` unexpected error.

## cleanup --dry-run

Purpose: Builds a cleanup plan without changing services, processes, files, registry, MSI state, or Defender state.

Destructive: No, except report writing.

Admin required: No, but missing admin is reported as a blocker for real cleanup.

Required inputs: None.

Writes: `initial-detection.json`, `cleanup-plan.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe cleanup --dry-run
```

Expected exit codes: `0` planned, `1` planned with warnings, `7` safety validation failure, `10` unexpected error.

## cleanup --yes

Purpose: Executes a validated cleanup plan.

Destructive: Yes.

Admin required: Yes.

Required inputs: `--yes` for non-interactive confirmation.

Writes: `initial-detection.json`, `final-detection.json`, `cleanup-plan.json`, `cleanup-result.json`, `rollback-info.json`, `operations.json`, `summary.txt`, `repair.log`, and possibly `msi-uninstall.log`.

Example:

```powershell
.\kigrepair.exe cleanup --yes
```

Expected exit codes: `0` success, `1` partial success or warnings, `2` admin required, `3` confirmation missing, `5` cleanup failed, `7` safety validation failure, `10` unexpected error.

## install

Purpose: Installs Grabber from a validated MSI package.

Destructive: Yes, because MSI install changes system state.

Admin required: Yes.

Required inputs: `--installer`, `--invite`, `--yes`.

Writes: `install-result.json`, `installer-validation.json`, `final-detection.json`, `operations.json`, `summary.txt`, `repair.log`, `msi-install.log`.

Example:

```powershell
.\kigrepair.exe install --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Expected exit codes: `0` success, `1` warnings, `2` admin required, `3` confirmation missing, `7` validation or MSI failure, `10` unexpected error.

## defender status

Purpose: Reads Defender status and exclusion coverage.

Destructive: No, except report writing.

Admin required: No, but policy or Defender availability can limit results.

Required inputs: None.

Writes: `defender-result.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe defender
```

Expected exit codes: `0` success, `1` warnings, `10` unexpected error.

## defender ensure

Purpose: Adds missing Defender exclusions for detected or configured Grabber paths.

Destructive: Yes.

Admin required: Yes.

Required inputs: `--ensure`, `--yes`. Use `--all-known-paths` only when support policy approves adding all configured paths.

Writes: `defender-result.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe defender --ensure --yes
```

Expected exit codes: `0` success, `1` warnings, `2` admin required, `3` confirmation missing, `7` Defender ensure failed, `10` unexpected error.

## repair --dry-run

Purpose: Builds the repair execution plan without modifying the system.

Destructive: No, except report writing.

Admin required: No, but missing admin is reported as a blocker for real repair.

Required inputs: `--installer`, `--invite`.

Writes: `repair-plan.json`, `preflight-result.json`, `initial-detection.json`, `cleanup-plan.json`, `classification-result.json`, `recommendation-result.json`, `installer-validation.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
```

Expected exit codes: `0` planned, `1` planned with warnings, `7` not ready, `10` unexpected error.

## repair --yes

Purpose: Runs full repair: detect, snapshot, cleanup, install, Defender ensure, final verification, and reporting.

Destructive: Yes.

Admin required: Yes.

Required inputs: `--installer`, `--invite`, `--yes`.

Writes: `repair-result.json`, `repair-plan.json`, `preflight-result.json`, `initial-detection.json`, `final-detection.json`, `cleanup-plan.json`, `cleanup-result.json`, `install-result.json`, `defender-result.json`, `verification-result.json`, `classification-result.json`, `recommendation-result.json`, `installer-validation.json`, `rollback-info.json`, `operations.json`, `summary.txt`, `repair.log`, `msi-install.log`, and possibly `msi-uninstall.log`.

Example:

```powershell
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
```

Expected exit codes: `0` success, `1` warnings, `2` admin required, `3` confirmation missing, `7` validation, cleanup, install, Defender, or verification failure, `10` unexpected error.

## collect-report

Purpose: Collects read-only diagnostics and creates `kigrepair-support-bundle.zip`.

Destructive: No, except report writing.

Admin required: No, but elevated access can improve event log and system collection.

Required inputs: None.

Writes: `collect-result.json`, `summary.txt`, `repair.log`, `kigrepair-support-bundle.zip`.

Example:

```powershell
.\kigrepair.exe collect-report
```

Expected exit codes: `0` success, `1` warnings or partial collection, `10` unexpected error.

## reports list

Purpose: Lists report directories under the configured report root.

Destructive: No.

Admin required: No, but access to `C:\ProgramData\kigrepair\Reports\` may depend on permissions.

Required inputs: None.

Writes: No workflow report by default.

Example:

```powershell
.\kigrepair.exe reports list
```

Expected exit codes: `0` success, `1` warnings, `10` unexpected error.

## reports cleanup --dry-run

Purpose: Creates a safe report cleanup plan without deleting report directories.

Destructive: No, except report writing.

Admin required: Depends on report directory permissions.

Required inputs: None.

Writes: `report-cleanup-plan.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe reports cleanup --dry-run
```

Expected exit codes: `0` planned, `1` planned with warnings, `7` safety validation failure, `10` unexpected error.

## reports cleanup --yes

Purpose: Deletes only validated old report directories inside the report root.

Destructive: Yes.

Admin required: Depends on report directory permissions.

Required inputs: `--yes`.

Writes: `report-cleanup-plan.json`, `report-cleanup-result.json`, `operations.json`, `summary.txt`, `repair.log`.

Example:

```powershell
.\kigrepair.exe reports cleanup --yes
```

Expected exit codes: `0` success, `1` partial success or warnings, `3` confirmation missing, `7` safety validation failure, `10` unexpected error.
