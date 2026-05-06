# Configuration

`kigrepair` works without a config file. Built-in defaults remain safe and CLI flags override configured values.

## Lookup Order

1. `--config "C:\Path\kigrepair.yaml"`
2. `<exe-dir>\kigrepair.yaml`
3. `C:\ProgramData\kigrepair\kigrepair.yaml`
4. `.\kigrepair.yaml`

## Commands

```powershell
.\kigrepair.exe config sample
.\kigrepair.exe config validate --config ".\kigrepair.yaml"
.\kigrepair.exe config show --config ".\kigrepair.yaml"
.\kigrepair.exe config show --config ".\kigrepair.yaml" --json
```

## Profiles

- `standard`: balanced support defaults. Real repair is allowed only after normal gates pass, rollback is required, Defender coverage is required for install when available, known cleanup targets are allowed, and reports retain older than 30 days while keeping the newest 10.
- `conservative`: stricter defaults for uncertain machines. It blocks unknown detection for repair, treats Defender unavailability more cautiously, skips hidden WMI cleanup by default, uses stricter installer metadata policy, and retains older than 60 days while keeping the newest 20.
- `diagnostic`: read-only oriented defaults. Logging defaults to debug, Defender ensure is not automatic, the wizard does not offer repair unless explicitly requested, diagnostics are broader, and reports retain older than 14 days while keeping the newest 30.

Use a CLI profile override when support needs a different strategy for one run:

```powershell
.\kigrepair.exe check --profile conservative
.\kigrepair.exe verify --profile diagnostic
.\kigrepair.exe preflight --profile conservative --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --profile conservative --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe collect-report --profile diagnostic
```

## Override Precedence

1. CLI flags, such as `--output`, `--installer`, `--quiet`, and `--json`
2. Values from `kigrepair.yaml`
3. Built-in selected profile
4. Hardcoded safe defaults

`--profile standard|conservative|diagnostic` overrides `profile:` in YAML. Command flags override the effective policy where that command supports an override.

## Hard Safety Rules

Config validation rejects attempts to disable hard safety gates:

- administrator requirement for destructive actions
- `--yes` requirement for quiet or non-interactive destructive actions
- rollback snapshot requirement before mutation
- exact cleanup allowlist requirement
- revalidation before mutation

Profiles cannot weaken hidden WMI process validation, cleanup path allowlisting, rollback snapshot creation, or service/process trust validation.

## Forbidden Secrets

Config files must not contain invite values, passwords, tokens, authorization headers, or secrets. Pass invites only at runtime with `--invite`.

Forbidden key names include `invite`, `password`, `token`, `access_token`, `refresh_token`, `secret`, and `authorization`.

Workflow service request structs may hold an invite in memory for one run, but invite fields are non-serializable and must never be stored in config, logs, reports, or support bundles.

## Sample

See [examples/kigrepair.sample.yaml](../examples/kigrepair.sample.yaml).
