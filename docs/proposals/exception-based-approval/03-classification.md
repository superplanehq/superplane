# Classification

The design rests on the category being trustworthy. A dangerous change mislabeled
low risk is exactly the catastrophe the ceiling exists to prevent. So
classification is deterministic, legible, and conservative.

Implemented in `pkg/autoapprove/classify.go`.

## Category and tier are separate

- **Category** is the class of change: docs, tests, config, dependency, app code,
  migration, auth, secrets, infra.
- **Tier** is the risk level: low, mid, or high. Only low is ever eligible for
  automatic approval.
- **Observed behavior** is what actually happened to changes of that category over
  time. It is recorded in the ledger and used by the later rungs. It can disagree
  with the declared tier, which is a feature: it tells you where your sense of
  risk was wrong.

## Three properties that make it safe

**Fail-closed.** A change whose files cannot be read from the payload, or that
matches no category, is treated as high risk, never low. Unknown is dangerous.

**Danger floors.** Certain path signals force a change to high no matter what else
it contains: migrations, auth and authorization, secrets, and production
infrastructure. If any path in the change hits a floor, the whole change is high.
This is why a dangerous change cannot be hidden among safe files, and why a user
who sets categories cannot relabel a production deploy as low risk. The floor is
enforced in code, not in configuration.

**Inert is a strict subset of low.** Only documentation and no-op changes are
inert. A low-risk change that is not inert, such as a tests-only change, still
requires a human in Stage 1. Inert is the smallest, safest set, and it is all that
clears automatically today.

The test `TestClassify_FloorsCannotBeDilutedBySafeFiles` asserts the floor
property directly: a migration, an auth file, a secret, or an infra file mixed in
with any number of safe files is still classified high.
