# Ordered pull request review checks

`github.setPullRequestReviewCheck` is a reusable canvas action. It does not
select reviewers or apply Factory policy. Review request uses this action when Publish PR check is enabled. The setting
is off by default.

## Component contract

| Field | Meaning |
| --- | --- |
| `repository`, `pullNumber` | Repository and pull request. Short repository names use the integration owner. |
| `assessedSha` | Commit covered by this decision. |
| `context` | Stable GitHub commit status context. |
| `sourceRunId` | Source run. Defaults to the persisted broadcast source, or the current run for ordinary triggers. |
| `attemptRootEventId` | Attempt root event. Defaults to the current execution root event. |
| `activationGeneration` | Positive saved activation version. After disabling, select a greater generation to enable again. |
| `enabled` | Defaults to true for ordinary check nodes. Saving false releases that node's managed checks. |
| `activationNodeId` | Optional check node in the same canvas and integration. Its admitted enabled state and generation govern participation. |
| `phase` | `pending_selection`, `required_reviewers`, `no_review`, `selection_error`, or `disabled`. |
| `reviewers` | Required human reviewer IDs and logins. Required for `required_reviewers`. |
| `reviewerRequestExecutionId` | Recorded Ensure Pull Request Reviewers execution. Required for `required_reviewers`. |
| `reason` | Reason for the decision. |
| `resultUrl` | Optional status details link. |

GitHub treats status contexts as
[case-insensitive](https://docs.github.com/en/rest/commits/statuses#create-a-commit-status).
The action stores their lowercase form so case variants share one requirement.

Ensure Pull Request Reviewers emits `executionId` with its existing outcome.
Pass that ID to this action. The referenced execution must belong to the same
canvas, attempt, integration, repository, and pull request. Its successful
output must cover the assessed commit and the same reviewer identities.
The check action does not request reviews again.

`github.pullRequestReviewCheck` uses the default output channel:

```json
{
  "type": "github.pullRequestReviewCheck",
  "data": {
    "status": "applied",
    "state": "pending",
    "sha": "current-head-sha",
    "phase": "required_reviewers",
    "context": "team/review",
    "generation": 2,
    "activationGeneration": 1,
    "sourceRunId": "source-run-id",
    "attemptRootEventId": "attempt-root-event-id"
  }
}
```

`status` is `applied` after delivery, `superseded` when newer state wins, or
`skipped` when the admitted attempt does not participate. Skipping creates no
publication record.
The action finishes after delivery. Approval tracking continues independently.

## Status behavior

All required reviewers must approve the current assessed commit. Existing
approvals count. The client reads every review page. Comments do not change
approval. Dismissal revokes approval. A dismissed replacement does not restore
an older approval.

Selection and missing approvals produce pending. A selected changes request
produces failure. Selection errors and provider lookup failures produce error.
All required approvals, explicit no-review decisions, and disabling produce
success. Disabling uses the description `Review request check disabled`.

A new head produces pending until a new decision covers it. Monitoring
continues after success. Closed or merged pull requests stop polling after
delivery. Pull request events resume validation on reopening.

## Ordering and authorization

One current record exists per integration, repository, pull request, and context.
Ordering uses persisted values in this order:

1. Source run creation time, then source run ID.
2. Attempt root event creation time, then root event ID.

Agent timestamps and delivery times have no authority. An older source cannot
replace a newer source, even when its retry finishes later. Newer attempts of
the same source supersede older attempts, including on the same commit.

Broadcast delivery stores source provenance outside the immutable message
payload. Explicit Runs retries copy that provenance and the original payload.
The model verifies the current organization, integration, canvas, source run,
and attempt. Callers cannot substitute unrelated run references.

Within one attempt, pending selection can become one outcome. Disable can
supersede that outcome. The disabled record remains as a fence. Work from that
activation cannot reopen it, even with a newer source. Re-enabling requires
a greater activation generation and a current or newer source and attempt.
A PR can miss intermediate activations. A new PR can start in any positive generation.

Each accepted replacement increments `generation`. Each persisted update
increments `revision`. Refresh checks the revision after lookups and claims
delivery before publishing. Concurrent refreshes share one delivery lease.

## Reuse without a check

`core.ExecutionContext.Publications` exposes `ResolveOrder`, `ExecutionOutput`,
and the `PublicationStore` methods. Findings publication can use these APIs
without a status, review monitoring, or an active check node.

Resolve order from the current execution. Use a separate, stable publication
key for findings. In `Update`, call `OrderedPublication.CanReplace` with the
resolved order and activation generation before replacing the current state.
Persist the accepted order and generation with the component-owned payload.
An empty `Hook` does not schedule work.

The store treats payloads as opaque JSON. It does not interpret reports,
recognize templates, or invoke named automation callbacks. Phase 1 canvases
remain usable without managed publication records.

## Durable delivery

`Update` holds a lock for one integration and publication key. It persists the
record and its next integration request in one transaction. Matching requests
form one recurring chain.

Refresh reads GitHub outside the transaction. A short transaction checks the
revision, claims a one-minute delivery lease, and schedules recovery. The status
write runs outside the transaction with a 20-second deadline measured from the
claim. A short transaction then records the result and releases the lease.

Replacement and disable can save newer decisions while delivery runs. They
preserve the active lease so another status write waits for completion or lease
expiry. An older delivery cannot acknowledge a newer decision. Completion
schedules immediate refresh when the revision changes. A refresh that finds an
active lease preserves recovery work for lease expiry.

Failed or interrupted delivery leaves recovery work pending. Expired leases
permit recovery after worker interruption. A late response from an expired
owner cannot release a newer lease. It schedules another delivery of the current
decision, including for disabled or closed checks.

Delivery is at least once. A crash after GitHub accepts a status but before
the completion transaction can repeat that status. Duplicate successful
application does not issue another reviewer request or status write.

The integration request worker retries failed hooks with bounded backoff.
After ten failures, it retains a failed request and its error. A leased request
can recover after worker interruption. Existing recurring recovery requests
remain independent of the failed request.
Reapplying an undelivered requirement restarts delivery without changing its
generation.

Pending selection checks inspect their owning attempt during refresh.
Cancellation, failure, or completion without a valid decision produces error,
including failures that do not emit a failed-channel event.

## Deployment

Apply the generated migration before deploying API and worker code. Both
processes require the existing database, encrypted integration credentials,
and `ENCRYPTION_MASTER_KEYS`. Workers use the existing integration-request,
node-request, and webhook workers. Runners require no new configuration.

GitHub needs existing commit-status write and pull-request read permissions.
Reviewer requests continue through the existing reviewer action. Repository
webhook provisioning uses the existing repository-hook permission. This change
adds no Checks API permission, service, secret, environment key, or branch rule.

Local Docker and the repository chart provide the existing worker and
encryption configuration. Hosted chart and launchpad verification was
unavailable through the connected GitHub account. No companion deployment PR
is required by a new configuration key. Deployed configuration remains
unverified. Do not enable the later Review request phase until combined
Phase 2 validation passes.

## Review request configuration

Publish PR check is off for new and existing automations. Reviewer selection
and reviewer requests continue independently. Save the switch to publish
`superplane/review-request` for future attempts. Saving does not start an agent,
backfill decisions, or attach a check to an attempt that already started.

The existing canvas settings store `publishPRCheck` and
`prCheckActivationGeneration` on `select-reviewers`. The `pr-check-pending`
node stores the corresponding `enabled` and `activationGeneration` values.
An off-to-on save increases the generation. Other saves retain it. Admin reset
preserves both settings, fallback, models, and Additional instructions,
including an explicitly cleared value.

The check action reads participation from the attempt's admitted canvas
version. At application and refresh, it verifies that the live activation
still matches. Disabling durably releases existing requirements with success
and the reason `Review request check disabled`. Failed release delivery retries
through integration requests. Re-enabling cannot resume old callbacks.
Removing a canvas or its activation node durably wakes owned publications so
the same release path also handles checks that stopped polling on closed PRs.
Pausing changes trigger admission and keeps existing approval tracking.

The canvas registers pending before candidate discovery. It applies explicit
no-review outcomes for all-positive reports and empty eligible candidate lists.
Empty lists skip the selection agent. Required-reviewer application uses the
recorded Ensure reviewers execution and does not request reviewers again.
Stale or skipped outcomes cannot publish no-review success. Failed or cancelled
attempts without a valid decision produce error through the pending check's
existing refresh hook.

Explicit Runs retry creates a new attempt of the original report. It uses the
current saved activation, current-SHA checks, and persisted source/attempt
ordering. It does not load the latest report. Review and dismissal webhooks
refresh approval state independently of the completed selection run.

Administrators must require `superplane/review-request` in GitHub branch rules
to enforce required approvals. SuperPlane does not modify branch rules.

Before enabling publication on an older installed canvas, use the existing
admin reset to install the new explicit check steps. The settings editor keeps
ordinary saves available for older canvases and explains this prerequisite if
publication is requested. Reset preserves supported settings and does not
start selection or backfill decisions.

Apply Phase 2C's migration before API and worker rollout. This phase adds no
migration or runtime configuration key. API and workers continue to require
`ENCRYPTION_MASTER_KEYS` and the existing encrypted GitHub integration.
Workers require `START_INTEGRATION_REQUEST_WORKER`, `START_NODE_REQUEST_WORKER`,
`START_APP_MESSAGE_WORKER`, and `START_WEBHOOK_PROVISIONER`. Runners require
no change. Local Docker and the repository chart include these settings.
The hosted chart and launchpad repositories returned HTTP 404 through `gh`.
Their current configuration and deployed configuration remain unverified.
