# Technical design

The change reuses machinery already in the codebase. It adds one pure package and
a small, additive branch on the existing `approval` component.

## Layout

- `pkg/autoapprove/`, the pure decision logic, with no dependency on the rest of
  the platform, so it is fully unit-testable on its own.
  - `change.go`, normalizes an event payload into a provider-agnostic `Change`.
  - `classify.go`, categories, tiers, danger floors, fail-closed defaults.
  - `policy.go`, `Decide`, the single place the ceiling is enforced.
  - `ledger.go`, the `DecisionRecord` written to canvas memory.
- `pkg/components/approval/autoapprove.go`, the wiring: the `AutoApprovePolicy`
  config, the `tryAutoApprove` method, and the optional configuration field.
- `pkg/components/approval/approval.go`, three additive edits (a config field, a
  branch at the top of `Execute`, and one appended configuration field).

## How a decision is made

1. `Execute` decodes the config. If the gate has no `AutoApprove` policy, nothing
   changes and the manual flow runs exactly as before.
2. `tryAutoApprove` builds a `Change` from `ctx.Data`, classifies it, and, if a
   guard expression is set, evaluates it through `ctx.Expressions.Run`. This is the
   same `expr-lang` evaluator the `filter` and `if` components use, exposed through
   the `FieldTypeExpression` field.
3. `Decide` applies the ceiling first: anything above low is escalated, ignoring
   every other setting. Below that, an inert low-risk change is auto-approved only
   when the instance switch is on, the gate opts in, and the guard matches.
4. Every decision is appended to the ledger through `ctx.CanvasMemory.Add`. No new
   database schema is needed.
5. On auto-approval, the gate writes an approval `Record` of type `auto` and emits
   on the existing `approved` channel. The record shares the shape of a human
   approval, so the audit trail is uniform.

## One audit trail

The auto record uses the existing `Record` and `ApprovalInfo` types with a new
`Type` value, `auto`, and the decision reason in the comment. Nothing downstream
that reads approval records needs to change, and the audit trail now answers "who
or what approved this, and why." This feeds, rather than competes with, the
tamper-evident audit-log request in
[#1565](https://github.com/superplanehq/superplane/issues/1565).

## Outcome binding (Stage 2)

Each `DecisionRecord` carries a correlation key taken from the payload (a commit
SHA, a pull-request number, a deploy id) or the execution id. A later revert,
rollback, failed post-merge CI run, or incident updates the matching record by
key. SuperPlane can do this because it orchestrated the deploy and is wired to the
incident and version-control tools: the decision and its outcome live in the same
execution graph. No standalone code-review tool holds both ends.

## Testing

The decision logic in `pkg/autoapprove` is covered by table-driven tests with
synthetic payloads and runs without any infrastructure, including the floor and
ceiling properties. The component wiring is covered in
`pkg/components/approval/autoapprove_test.go` against the standard test contexts.
