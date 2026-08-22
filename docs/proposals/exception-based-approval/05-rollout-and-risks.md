# Rollout and risks

## Three stages

**Stage 1 (this PR).** Classify every change on a gate that has a policy, record
every decision, and auto-clear inert low-risk changes behind an off-by-default
instance switch. Human approval is unchanged for everything else. Testable with
synthetic payloads, no agent required.

**Stage 2.** Bind outcomes into the ledger. Correlate reverts, rollbacks, failed
post-merge CI, and incidents back to their decision records, using orchestration
events the platform already emits. Produces the per-category report of frequency
and impact.

**Stage 3.** The two human rungs. An organization-scoped permission to unlock a
category, a canvas-author opt-in to turn it on for an app, and the console report
that surfaces each category's behavior with sample-size confidence, so no category
is freed on a handful of lucky runs.

## Where it is hard, stated plainly

- **The outcome signal is a noisy proxy.** "No incident in 48 hours" does not
  prove a change was good. An outcome is never a verdict, only evidence that nudges
  a category's record. A pattern moves it, not a single event.
- **Attribution is genuinely hard.** Several changes deploy together and one
  triggers a rollback. Attribute to the window and stay conservative; a team that
  wants finer blame deploys in finer slices. Do not fake precision.
- **Feedback is delayed and out of order.** The consequence of a decision lands
  minutes to days later. The ledger reconciles late outcomes rather than assuming
  they arrive promptly.
- **Small samples lie.** "Docs: three changes, zero failures" is evidence of
  nothing. Unlock decisions in Stage 3 are sample-size aware; a category earns
  eligibility on volume, not on luck.

None of these are blockers. They are the design. Naming them is the difference
between a real control surface and a demo that looks clever until it flaps.

## Compatibility

Every new behavior is off by default. A gate with no `AutoApprove` policy is
byte-for-byte unchanged. The instance switch defaults off. No existing
installation changes behavior on upgrade.
