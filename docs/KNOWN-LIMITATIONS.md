# Known Limitations

This file records release-known limitations that support engineers should understand before using `kigrepair`.

## Current Limitations

- GUI is internal MVP only; reports remain the source of evidence.
- Third-party antivirus behavior may vary by vendor, policy, and endpoint state.
- Defender policy-managed environments may block exclusions or make exclusion state read-only.
- Rollback snapshot is audit-only; it records pre-mutation state but does not automatically restore the endpoint.
- Hidden WMI repair requires exact validation of service image paths and executable paths before mutation.
- Real VM coverage may still be incomplete for some customer environments and must be tracked in release evidence.
- MSI is not bundled by default; support must provide a validated installer path.
- Non-admin runs can perform read-only checks, but real repair remains blocked before mutation.

## Not Limitations

Do not hide release blockers in this file. Any open P0 or P1 issue from [Release Blockers](RELEASE-BLOCKERS.md) or [Bug Burndown](BUG-BURNDOWN.md) must remain a blocker until fixed or formally waived outside this repository.

## Validation Expectations

Before pilot, complete the minimum gates in [VM Release Gates](vm-testing/VM-RELEASE-GATES.md). Before internal-stable, complete required Tier 2 cases or document waivers in the release ticket.

