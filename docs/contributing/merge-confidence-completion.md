# Merge confidence completion event

The GitHub merge-confidence app broadcasts `merge-confidence.completed` after
PR-body publication returns `published` or `unchanged` for a complete report.
Incomplete reports still reach the PR body, but do not emit completion.
Publication eligibility checks exclude disabled, cancelled, closed, stale,
superseded, and rejected outcomes. Failed assessments cannot emit completion
or become successful because a downstream action succeeds.

The app uses the existing `filter` and `broadcastMessage` actions. It does not
request reviewers, change checks, or add a consumer. Bitbucket keeps its
existing publication behavior and shared report format.

## Payload contract

The `onBroadcast` trigger receives the existing `app.broadcast` envelope.
The message worker supplies `data.app` and `data.node` from the source canvas
and node records. `data.payload` contains these fields:

| Field | Meaning |
| --- | --- |
| `type` | Always `merge-confidence.completed`. |
| `reportId` | Producer canvas run ID. This is the report identity. |
| `pullRequestId` | Factory PR ID from the producer activity and persisted PR. |
| `repository` | Provider repository name from the persisted PR. |
| `pullNumber` | Provider PR number from the persisted PR, as a JSON number. |
| `reviewedCommit` | Assessed SHA from the producer root event. |
| `mergeConfidence` | Exact immutable report from the assessment output event. |

The immutable report includes `repository` and `pullNumber` with its existing
PR identity, completeness, ratings, checks, assessed SHA, and task URL.
The producer does not read mutable latest-check rows or parse PR-body text.
It reads check report events scoped to the producer run when assessment ends.

Publication retries retain the producer run ID and assessment output.
A new producer run has a new report ID, even for the same assessed SHA and
identical report content. A retry can return `unchanged` after a provider
accepted the first write but the producer did not record success.

Example event received by a subscriber:

```json
{
  "type": "app.broadcast",
  "timestamp": "2026-11-11T18:00:00Z",
  "data": {
    "app": {
      "id": "11111111-1111-4111-8111-111111111111",
      "name": "Merge confidence"
    },
    "node": {
      "id": "broadcast-merge-confidence",
      "name": "Broadcast Merge Confidence"
    },
    "payload": {
      "type": "merge-confidence.completed",
      "reportId": "22222222-2222-4222-8222-222222222222",
      "pullRequestId": "33333333-3333-4333-8333-333333333333",
      "repository": "acme/app",
      "pullNumber": 42,
      "reviewedCommit": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
      "mergeConfidence": {
        "pullRequestId": "33333333-3333-4333-8333-333333333333",
        "repository": "acme/app",
        "pullNumber": 42,
        "complete": true,
        "overallScore": 4,
        "overallLevel": "positive",
        "checks": [
          {
            "id": "security",
            "label": "Security",
            "score": 4,
            "maxScore": 5,
            "summary": "No security gap.",
            "level": "positive",
            "missing": false
          }
        ],
        "reviewedCommit": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
        "taskUrl": "https://app.superplane.com/44444444-4444-4444-8444-444444444444/workspaces/eng-k7m2xqab/work-order/42"
      }
    }
  }
}
```

## Consumer freshness and delivery

Consumers must reload authoritative records using the source app, producer
run ID, and Factory PR ID. Validate their ownership and linkage before use.
Verify the payload against the persisted assessment and publication outcome.
Recheck the current PR state, head SHA, and supported settings before acting.

Order source runs by persisted `workflow_runs.created_at DESC`, then
`workflow_runs.id DESC`. This matches `FactoryPullRequest.ListActivityRuns`.
A newer applicable source run supersedes an older report, including for the
same SHA. Payload timestamps, delivery order, and completion order do not
establish freshness. The envelope timestamp describes delivery, not source
run creation. Do not use it to order reports.

The existing app-message worker delivers subscriber events in one transaction.
A delivery failure rolls back subscriber inserts and retains the pending
message. Its next poll retries delivery. Processing a consumed message does
not create duplicate subscriber events. Consumers should use `reportId` for
report-level idempotency.

## Installation and deployment

New materializations install the completion path. Admin reset installs it in
existing GitHub merge-confidence apps while preserving supported settings,
including publication opt-out, checks, custom check instructions, and agent
settings. Reset does not activate a Review request feature.

The API publication dispatcher and canvas workers use existing database and
integration configuration. Delivery requires `START_APP_MESSAGE_WORKER=yes`
on the process that runs `AppMessageWorker`. Runners need no new configuration.
This change adds no service, endpoint, credential, or Secret key.

Local Compose enables the worker in the combined app process. The bundled
Helm chart enables it in the worker deployment. Keep this worker enabled
before installing or resetting the updated template. No companion deployment
change is required for these checked sources.

Hosted deployments use the separate `superplanehq/helm-chart` and
`superplanehq/launchpad` repositories. Verify their environment values and
Secret references before rollout. Source configuration does not establish
the published chart version or the live worker setting.
