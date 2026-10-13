# Exception-based approval

**Make approval the exception, not the toll booth. Auto-clear what is provably safe, keep a human on everything that matters, and record every decision either way.**

Your users are already telling you where SuperPlane slows down. Parked approval
runs pile up ([#6405](https://github.com/superplanehq/superplane/issues/6405)),
and the request on the table is to manage that queue better
([#1614](https://github.com/superplanehq/superplane/issues/1614)). Both make the
queue nicer. Neither makes it smaller.

This proposal changes the shape of the problem instead. The provably safe changes
never enter the human queue, and by construction nothing that could hurt you can
clear without a person. As the number of agent-driven changes grows, the reviewer
stops being the throughput limit.

This PR is Stage 1. It classifies every change an approval gate sees, records the
decision in an append-only ledger, and auto-clears only the inert cases, and only
when an operator turns the feature on. Mid and high risk always require a human,
and no setting can override that.

## Read in order

1. [The bottleneck](01-the-bottleneck.md), why approval, not execution, is the next constraint.
2. [The trust ladder](02-trust-ladder.md), how autonomy is earned across three rungs.
3. [Classification](03-classification.md), what counts as safe, and why it cannot be gamed.
4. [Technical design](04-tech-design.md), how it is built, against the existing component model.
5. [Rollout and risks](05-rollout-and-risks.md), the three stages, and the honest hard parts.

## What is in this PR

- Classification of every change on a gate that has a policy, with system-enforced
  risk floors and a fail-closed default.
- One audit record per decision, automatic or escalated, so "what cleared without
  a person, and why" is answerable.
- Rung 0: automatic clearing of inert, low-risk changes only (documentation,
  no-op), behind the `AUTO_APPROVE_INERT_ENABLED` instance switch, off by default.
- Tests: the decision logic in `pkg/autoapprove` is fully unit-tested with
  synthetic payloads and needs no running agent; the component wiring is covered
  in `pkg/components/approval/autoapprove_test.go`.

A gate with no auto-approval policy behaves exactly as it does today.
