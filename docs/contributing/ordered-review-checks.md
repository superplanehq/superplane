# Ordered pull request review checks

`github.setPullRequestReviewCheck` is a reusable canvas action. It does not
select reviewers or apply Factory policy. Review request does not use this
action until the later activation phase.

## Component contract

| Field | Meaning |
| --- | --- |
| `repository`, `pullNumber` | Repository and pull request. Short repository names use the integration owner. |
| `assessedSha` | Commit covered by this decision. |
| `context` | Stable GitHub commit status context. |
| `sourceRunId` | Source run. Defaults to the persisted broadcast source, or the current run for ordinary triggers. |
| `attemptRootEventId` | Attempt root event. Defaults to the current execution root event. |
| `activationGeneration` | Starts at 1. After disabling, explicitly select the next generation to enable the check again. |
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

`status` is `applied` after delivery, or `superseded` when newer state wins.
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
the next activation generation and a current or newer source and attempt.

Each accepted replacement increments `generation`. Each persisted update
increments `revision`. Refresh checks the revision after lookups and before
delivery. Concurrent refreshes cannot publish an obsolete snapshot.

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

Refresh reads GitHub outside the lock. It then persists desired status and
recovery work before delivery. The final revision check and bounded status
write share the requirement lock. Replacement and disable use that same lock.
Failed or interrupted delivery leaves recovery work pending.

Delivery is at least once. A crash after GitHub accepts a status but before
the database commit can repeat that status. Duplicate successful application
does not issue another reviewer request or status write.

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
