# Internal Rollout

Use this document for internal support rollout planning. Release mechanics are covered in [Release Checklist](RELEASE-CHECKLIST.md), and real-Windows validation is covered in [VM Testing](vm-testing/README.md).

## Channels

- `developer`: local builds used by maintainers.
- `pilot`: limited support-engineer validation after MVP gates pass.
- `internal-stable`: broader internal support usage after expanded gates pass.

## Pilot Entry Criteria

- Build and release validation completed.
- VM pilot gates in [VM Release Gates](vm-testing/VM-RELEASE-GATES.md) passed or waived.
- Evidence archived in the internal release ticket.
- Known limitations reviewed and updated.
- Support runbook matches the shipped command set.
- No Grabber MSI or invite is bundled in the release.

## Internal-Stable Entry Criteria

- Pilot feedback reviewed.
- Required Tier 2 VM cases completed or formally waived.
- GUI smoke completed if GUI is distributed.
- Rollback and destructive-action blockers reviewed.
- Support escalation process is clear.

## Rollback

Keep the previous release zip, checksum, release manifest, and signatures record. If a release blocker is found after distribution, stop rollout, record the affected scenario ID, and revert support engineers to the previous release.

