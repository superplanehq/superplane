# Merge confidence assessment status

The GitHub Merge confidence canvas publishes the commit status
`superplane/merge-confidence`. It uses **Set Pull Request Check** before
assessment and after report publication. The status remains advisory.
Risk findings do not change a complete assessment from success to failure.

| Assessment outcome | Commit status |
| --- | --- |
| Active assessment | `pending` |
| Complete assessment and published or unchanged report | `success` |
| Failed or incomplete report, or skipped publication | `failure` |
| Attempt ends without a final result, including cancellation | `error` |
| Canvas removal, closed PR, changed commit, or superseded commit | `error` |

The generic GitHub component shares ordered publication records, delivery
leases, and integration requests with review checks. It does not create
reviewer requirements. It works in ordinary canvases without Factory records.
Persisted source-run and root-event identities determine ordering. A terminal
result cannot be reopened by a delayed start from the same attempt.

Desired state is persisted before delivery. Retries do not run assessment
agents or repeat completion broadcasts. Pending statuses on older commits
receive an error before the newer commit's status is delivered. Source history
cleanup does not delete publication records. Canvas removal resolves pending
delivery through the generic owner check. Deleted organizations cannot publish.

Missing integrations and permanent permission/resource failures stop polling
and record an error. Correct access and retry the publication step to resume.
Transient delivery failures use the existing integration request worker.
Delivery is at least once; an interrupted response can repeat an identical status.

Incomplete-report validation remains in `pkg/factories/mergeconfidence`.
Completion broadcasts retain their immutable payload and publication gates.
Review request instructions, reviewer requests, and completion delivery are unchanged.

API and workers use existing database and encrypted integration credentials,
including `ENCRYPTION_MASTER_KEYS` and existing `SUPERPLANE_GITHUB_APP_*`
configuration where applicable. Workers require
`START_INTEGRATION_REQUEST_WORKER=yes` and `START_NODE_REQUEST_WORKER=yes`.
Runners require no new configuration. Apply the existing ordered-publication
migration from #9297 before deploying this code. No new migration, Secret,
permission, chart change, or branch rule is required.

New installations use the updated canvas. For an existing installation, reset
the Merge confidence automation through the existing admin reset action.
The reset preserves supported saved check and model settings.

Local Docker and the repository Helm chart configure both workers. The connected
GitHub account could not read the hosted chart or Launchpad sources; their
configuration and deployed credentials remain unverified.

This implements #9278 on the updated #9246 branch. The Phase 3 PRs for #9259
and #9274 must merge before Phase 4 merge and combined validation. This branch
does not include their unmerged configuration or findings changes.
