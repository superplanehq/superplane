# SuperPlane — Pre-Improvement Diagnostics

Snapshot of the deep review performed on 2026-04-28. This is the
baseline-state report used to scope the improvement work tracked in
[PLAN.md](./PLAN.md). Findings here are deliberately broader than
what we plan to fix — anything not addressed here remains a
candidate for follow-up tasks.

## Top-level finding

The codebase is structurally sound. Issues are localized — primarily
in the integration layer (HTTP resilience, structured logging) and
in a small number of gRPC actions where transactional boundaries
are mishandled. There are exactly **8 TODOs** in `pkg/`, no FIXMEs
or HACKs.

---

## A. Transaction-correctness issues

The project's [AGENTS.md](../AGENTS.md) hard-rules: never call
`database.Conn()` inside a function that already participates in a
`*gorm.DB` transaction. Three locations violate this in spirit by
calling `authService.AssignRole` (which opens its own independent
GORM transaction via Casbin's gorm-adapter) from inside a
caller-side `Transaction(...)` block.

| Location | Issue |
|---|---|
| [pkg/grpc/actions/organizations/create_invitation.go:96-98](../pkg/grpc/actions/organizations/create_invitation.go#L96) | `// TODO: this is not using the transaction properly` directly above `authService.AssignRole(...)` inside a `database.Conn().Transaction(func(tx) error {...})` block |
| [pkg/grpc/actions/organizations/create_invitation.go:170-172](../pkg/grpc/actions/organizations/create_invitation.go#L170) | Same TODO — different code path (existing-account flow). Worse: the surrounding code uses manual `tx.Begin() / Rollback() / Commit()` instead of the closure-based helper, with explicit `Rollback()` calls strewn through the function body |
| [pkg/grpc/actions/organizations/remove_user.go:32](../pkg/grpc/actions/organizations/remove_user.go#L32) | `// TODO: this should all be inside of a transaction` |

Why it matters: if any step in the GORM transaction rolls back after
`AssignRole` succeeds, a stranded role grant remains in `casbin_rule`
and the user appears to have permissions on a resource that does not
exist (or has been deleted).

The Casbin enforcer is built once with
`gormadapter.NewTransactionalAdapterByDB(database.Conn())` at
[pkg/authorization/service.go:36](../pkg/authorization/service.go#L36),
and `AssignRole` itself wraps the policy mutation in its own gorm tx
via `adapter.Transaction(enforcer, fn)` at
[pkg/authorization/service.go:430](../pkg/authorization/service.go#L430).
That transaction is independent of the caller's tx, which is the
root cause.

## B. Outbound HTTP resilience gaps

`go.mod` has no resilience dependency (no `cenkalti/backoff`, no
`sony/gobreaker`, no `hashicorp/go-retryablehttp`). Outbound calls
in integrations are direct `http.Client.Do(req)` with no retry, no
backoff, and in some cases no timeout.

| Location | Issue |
|---|---|
| [pkg/integrations/slack/client.go:179](../pkg/integrations/slack/client.go#L179) | `client := &http.Client{}` constructed *per request* inside `execRequest`. No timeout — a hung Slack call holds a worker indefinitely. No connection reuse — TLS handshake on every call. No retry on 429/5xx |
| [pkg/integrations/aws/sns/on_topic_message.go:301](../pkg/integrations/aws/sns/on_topic_message.go#L301) | TODO: `it would be good to not fetch the certificate every time`. Webhook signature verification fetches the SNS signing certificate via HTTPS *on every webhook delivery* |
| Most other integrations | Spot-checked: PagerDuty, GitLab, Azure follow the same no-retry pattern. Not enumerated exhaustively |

## C. Webhook handler gaps

There is a clean central dispatch at
[pkg/public/server.go:1115](../pkg/public/server.go#L1115) and
[pkg/public/server.go:1150](../pkg/public/server.go#L1150) where
trigger and action `HandleWebhook` methods are invoked. Today, this
seam does not enforce idempotency: a replayed webhook (or one
delivered twice due to network retry) is processed twice and emits
duplicate events.

Most providers send a stable delivery identifier:
- AWS SNS: `MessageId` in the JSON body
- GitHub: `X-GitHub-Delivery` header
- Slack: `X-Slack-Retry-Num` (signal of replay) plus `Retry-Reason`
- PagerDuty: `X-PagerDuty-Signature` is per-payload but body has `id`
- Stripe-style: `id` in the event envelope

There is no `webhook_deliveries` (or similar) table in
`db/migrations/`.

## D. Context propagation gaps

Several integration call sites discard the caller's `ctx` and pass
`context.Background()` to outbound SDK calls — this breaks request
cancellation/deadline propagation.

- [pkg/integrations/gitlab/create_issue.go:225](../pkg/integrations/gitlab/create_issue.go#L225)
- [pkg/integrations/gitlab/run_pipeline.go:201](../pkg/integrations/gitlab/run_pipeline.go#L201)
- [pkg/integrations/gitlab/run_pipeline.go:255](../pkg/integrations/gitlab/run_pipeline.go#L255)
- [pkg/integrations/azure/component_restart_vm.go](../pkg/integrations/azure/component_restart_vm.go) — Azure VM operations

## E. Logging consistency

The repo standardizes on `logrus`, but stdlib `log` leaks into a
handful of integration files. The most visible offender is the
PagerDuty webhook path with 8+ `log.Printf` calls and no request
correlation IDs:

- [pkg/integrations/pagerduty/on_incident_status_update.go:145-200](../pkg/integrations/pagerduty/on_incident_status_update.go#L145)

Other stdlib `log` users: `pkg/models/canvas_node.go` (webhook
deletion), `pkg/server/server.go` (route registration).

## F. Loose typing

- 54 occurrences of `interface{}` in `pkg/` (AGENTS.md prefers `any`).
  Mechanical sweep — low value unless paired with another change.
- `_, _ = structpb.NewStruct(...)` in `pkg/grpc/actions/common.go`
  and `pkg/grpc/action_service.go` discards errors silently.

## G. TODO inventory (full)

Total: 8 in `pkg/`. None elsewhere of consequence.

**Transactions (3):**
- [pkg/grpc/actions/organizations/create_invitation.go:96](../pkg/grpc/actions/organizations/create_invitation.go#L96)
- [pkg/grpc/actions/organizations/create_invitation.go:170](../pkg/grpc/actions/organizations/create_invitation.go#L170)
- [pkg/grpc/actions/organizations/remove_user.go:32](../pkg/grpc/actions/organizations/remove_user.go#L32)

**Performance (2):**
- [pkg/integrations/aws/sns/on_topic_message.go:301](../pkg/integrations/aws/sns/on_topic_message.go#L301) — SNS cert caching
- `pkg/integrations/slack/slack.go` — unspecified TODO

**Behavior (3):**
- `pkg/grpc/actions/canvases/invoke_node_execution_hook.go`
- `pkg/grpc/actions/canvases/changesets/output_channels.go`
- `pkg/grpc/actions/canvases/changesets/canvas_publisher.go`

## H. Frontend (informational, out of scope for the planned phases)

- `key={index}` in [web_src/src/ui/componentBase/index.tsx:452](../web_src/src/ui/componentBase/index.tsx#L452) and [:486](../web_src/src/ui/componentBase/index.tsx#L486)
- File-wide `eslint-disable @typescript-eslint/no-explicit-any` in [web_src/src/ui/componentSidebar/pages/ExecutionChainPage.tsx:1](../web_src/src/ui/componentSidebar/pages/ExecutionChainPage.tsx#L1) and [web_src/src/ui/componentSidebar/SidebarEventItem/SidebarEventItem.tsx:1](../web_src/src/ui/componentSidebar/SidebarEventItem/SidebarEventItem.tsx#L1)
- Brand casing slip: `alt="Superplane"` in [web_src/src/ui/hoverCard/index.stories.tsx:36](../web_src/src/ui/hoverCard/index.stories.tsx#L36) — should be `"SuperPlane"`
- Missing `aria-label` on icon buttons in [web_src/src/ui/componentBase/index.tsx:343-385](../web_src/src/ui/componentBase/index.tsx#L343)
