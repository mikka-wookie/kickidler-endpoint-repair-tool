# VM Release Gates

## Minimum for Pilot

Must pass:

- Tier 0 smoke on at least one Windows 10 or Windows 11 VM.
- Tier 1 not-installed clean machine.
- Tier 1 healthy standard install.
- Tier 1 invalid installer blocks repair.
- Tier 1 missing invite blocks preflight/repair readiness.
- Tier 1 non-admin blocks real repair.
- Tier 1 support bundle redacts fake invite.
- Tier 1 normal Windows process false-positive protection.
- Release validation script passes.
- No raw invite found in reports or evidence.

Should pass:

- Defender parent exclusion covers child.
- Defender prefix false positive protection.
- Partial MSI leftovers detected.
- Partial files leftovers detected.

## Required Before Internal-Stable

- All pilot gates.
- Hidden WMI healthy if hidden mode is supported in field.
- Hidden WMI binary missing if setup is available.
- Service path mismatch is not mutated.
- Process path mismatch is not terminated.
- Rollback snapshot required before mutation.
- Rollback snapshot failure aborts mutation.
- Report cleanup safety.
- GUI MVP smoke if GUI is distributed.

## Release Blockers

Block release if any validation shows:

- Raw invite leak.
- Deletion outside allowlisted paths.
- Service/process mutation on path mismatch.
- Normal Windows process treated as Grabber.
- Unsupported MSI accepted as valid.
- Real repair runs without rollback snapshot.
- Destructive action runs without admin.
- Non-interactive destructive action runs without `--yes`.
- Support bundle contains secrets.
- GUI bypasses workflow service or safety gates.

## Waivers

Any skipped Tier 1 or required Tier 2 case needs:

- Scenario ID.
- Reason.
- Owner.
- Expiration or follow-up build.
- Release impact.
- Approval in the internal release ticket.

