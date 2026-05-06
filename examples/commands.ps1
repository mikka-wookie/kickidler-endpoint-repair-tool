.\kigrepair.exe version
.\kigrepair.exe version --json
.\kigrepair.exe check
.\kigrepair.exe verify
.\kigrepair.exe preflight --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe repair --dry-run --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>"
.\kigrepair.exe collect-report
.\kigrepair.exe reports list
.\kigrepair.exe config show

# Destructive. Use only after preflight and dry-run.
.\kigrepair.exe repair --installer ".\assets\grabberEM.x64.msi" --invite "<INVITE>" --yes
