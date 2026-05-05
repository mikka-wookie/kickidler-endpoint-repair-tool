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

- `standard`: normal support defaults.
- `conservative`: stricter repair safety, failed metadata validation and unavailable Defender status are treated more cautiously.
- `diagnostic`: read-only oriented defaults; the wizard does not offer repair unless explicitly requested.

## Override Precedence

1. CLI flags, such as `--output`, `--installer`, `--quiet`, and `--json`
2. Values from `kigrepair.yaml`
3. Built-in selected profile
4. Hardcoded safe defaults

## Forbidden Secrets

Config files must not contain invite values, passwords, tokens, authorization headers, or secrets. Pass invites only at runtime with `--invite`.

Forbidden key names include `invite`, `password`, `token`, `access_token`, `refresh_token`, `secret`, and `authorization`.

## Sample

See [examples/kigrepair.sample.yaml](../examples/kigrepair.sample.yaml).
