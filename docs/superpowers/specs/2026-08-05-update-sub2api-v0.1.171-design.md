# Sub2API v0.1.171 Update Design

## Goal

Update the local Recurdream API fork from its current Sub2API v0.1.170 base to upstream v0.1.171 while preserving all Recurdream customizations and the current balance-history work. Keep the result local until the user explicitly requests a GitHub push.

## Scope

- Preserve the red-packet activity, affiliate rebate and withdrawal flow, TTFT optimizer, profit control, Recurdream homepage, deployment diagnostics, and balance-history implementation.
- Import the complete upstream v0.1.171 source delta, including CAPTCHA providers, Codex identity/version synchronization, composite-group reasoning policies, reset-quota caching, refund confirmation, and upstream bug fixes.
- Update the server version and the Chinese main README to v0.1.171.
- Add a local annotated `v0.1.171` tag only after verification passes.
- Exclude `.superpowers/` session artifacts from every product commit.
- Do not push branches or tags and do not create a GitHub Release.

## Integration Strategy

Create a local checkpoint commit containing the completed product changes currently in the working tree, excluding `.superpowers/`. Create an isolated Git worktree from that checkpoint and perform the upstream update there so the primary checkout remains recoverable.

Use the official upstream v0.1.170 and v0.1.171 tagged source trees to derive the version delta. Apply that delta to the fork rather than replacing the repository wholesale. Resolve overlaps file by file, retaining local behavior unless v0.1.171 intentionally changes the same contract. Generated Ent and Wire files must remain consistent with their schemas and providers.

After the update commit and verification are complete, fast-forward the local `main` branch to the verified integration branch. Keep the checkpoint and integration commits in local history for the later combined GitHub push.

## Compatibility Decisions

- The v0.1.171 refund API behavior is authoritative: insufficient user balance returns `require_force`, and the administrator must explicitly confirm a forced refund.
- Existing Turnstile settings remain valid. Tencent Captcha and Alibaba Cloud Captcha 2.0 are added as mutually exclusive alternatives.
- Codex outbound identity normalization remains enabled by default, with `gateway.disable_codex_originator_normalization` available as the upstream rollback switch.
- Recurdream's local profit control and upstream billing-rate synchronization remain intact around the new Codex and scheduling behavior.
- Existing local migrations are never renumbered, deleted, or rewritten after application. New upstream migrations are retained under conflict-free filenames when necessary.
- Balance history remains a read-only aggregation over existing ledgers and must not be converted into a new accounting source.

## Release Metadata

Set `backend/cmd/server/VERSION` to `0.1.171`. Keep `README.md` Chinese-first and add a detailed v0.1.171 release section covering upstream features, breaking refund behavior, upgrade notes, preserved Recurdream features, and the balance-history addition when it is part of the checkpoint.

Do not replace the local README with the upstream English README. `README_CN.md` and `README_JA.md` may receive upstream changes only where they do not remove fork-specific documentation.

## Test Lifecycle Fix

The v0.1.170 compact-home test mounts `HomeView` wrappers without unmounting them. This leaves the API-address typewriter timer alive after Vitest tears down `window`. Track mounted wrappers and unmount them in test cleanup so the existing component `onBeforeUnmount` cleanup executes. This is a test-lifecycle correction and does not change the production homepage animation.

## Verification

- Verify the upstream tag/archive identity and inspect the v0.1.170-to-v0.1.171 change set before applying it.
- Check for unresolved conflict markers, whitespace errors, accidental deletions, and migration collisions.
- Run focused Go tests for changed backend packages, followed by the broad backend suite where practical.
- Run the full frontend Vitest suite and require a zero exit code after the HomeView cleanup.
- Run the frontend production build.
- Verify `VERSION`, README release text, the local tag target, and a clean product worktree except for ignored or intentionally untracked `.superpowers/` artifacts.

## Failure Handling

If upstream retrieval is interrupted, retry using the official release archive without altering the primary checkout. If a conflict cannot preserve both upstream behavior and a local customization, stop before committing that conflict resolution and report the exact files and behavioral choice required.

If a broad test fails for a pre-existing or environment-specific reason, retain the focused test evidence, document the failure precisely, and do not describe the repository as fully green.
