# internal/ui

`internal/ui` contains the GUI MVP boundary and Windows window code.

Rules:

- depend on `internal/app` workflow contracts
- do not import Cobra command packages
- do not shell out to `kigrepair.exe`
- do not parse CLI console output
- keep invite values masked, redacted, and cleared after runs where practical
- keep `internal/app` independent from this package

The Windows GUI code is behind `//go:build windows`; core controller and view model tests remain package-level safety coverage.
