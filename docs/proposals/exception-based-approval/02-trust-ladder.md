# The trust ladder

Autonomy is not granted in one step. It is earned across three rungs, and the
dangerous half of the range is never reachable at all.

## Rung 0: inert changes clear automatically

A conservative, built-in auto-approve for changes that are safe by construction
rather than by policy: documentation, no-op changes, changes that touch nothing
executable. This relieves the queue with nothing to configure per change. It is
system-defined, fail-closed, and behind a single instance switch
(`AUTO_APPROVE_INERT_ENABLED`) that is off by default, so an installation only
gets this behavior when an operator asks for it. Every automatic clearance is
still written to the audit trail.

This rung is what ships in this PR.

## Rung 1: the platform team unlocks a category

Every other category starts locked. The people who see the aggregate reports
review how a category has actually behaved and unlock the low-or-no-risk ones,
making them eligible. This rung decides whether a class of change is even allowed
to be automated. It maps to an organization-scoped admin permission in the
existing RBAC model. Mid and high risk never unlock.

## Rung 2: the app owner opts in

For a category the platform team unlocked, the owner of a specific app decides
whether to turn auto-approval on for their workflow, looking at the failure rate
and the impact those failures had. It maps to a canvas-author permission on the
app. Their call, their data, their app.

Rungs 1 and 2 arrive in Stage 3 (see [rollout](05-rollout-and-risks.md)). They
are described here so the shape of the whole design is clear.

## The ceiling

Low-risk categories can be freed. Mid and high risk cannot, ever, by any rung.
This is a product guarantee, not a tuning parameter: the system does not
auto-approve changes that touch migrations, auth, secrets, payments, or
production infrastructure. The worst case of the whole design is that too much
goes to humans, which is merely slow. It is never that something dangerous went
through, which is fatal.

The enforcement behind the ceiling is in [classification](03-classification.md):
danger signals are floors that no configuration can lower.
