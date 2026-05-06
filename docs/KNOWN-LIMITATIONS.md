# Known Limitations

This file records release-known limitations that support engineers should understand before using `kigrepair`.

## Current Limitations

- Real repair must be validated on disposable VMs before pilot use on support-owned endpoints.
- Hidden WMI scenarios require approved field-equivalent setup and may be waived when no approved test installer/state is available.
- Defender policy-managed and third-party AV behavior depends on lab availability.
- Non-admin runs can perform read-only checks, but real repair must be blocked before mutation.
- GUI MVP is an internal workflow shell; reports remain the source of evidence.

## Validation Expectations

Before pilot, complete the minimum gates in [VM Release Gates](vm-testing/VM-RELEASE-GATES.md). Before internal-stable, complete required Tier 2 cases or document waivers in the release ticket.

