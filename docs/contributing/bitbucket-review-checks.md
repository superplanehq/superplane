# Bitbucket managed review checks

`bitbucket.setPullRequestReviewCheck` works in ordinary canvases. It does not
enable the Review request automation switch or change repository rules.

## Component and shared contracts

The component accepts `repository` in workspace/repository format,
`pullRequestId`, a full `assessedSha`, `phase`, and `reason`. The repository
must be saved as a literal so Setup can provision its event subscription.
Other decision fields can use expressions. `resultUrl` defaults to the run.
The status key is always `superplane/review-request`.

Phases are `pending_selection`, `required_reviewers`, `no_review`,
`selection_error`, and `disabled`. Pending selection finishes after durable
registration. Other phases finish after delivery. Integration hooks continue
monitoring after the selection run finishes.

Required reviewers use `{uuid, displayName}` objects. UUIDs identify accounts;
display names have no authority. `reviewerRequestExecutionId` references the
same canvas, integration, and attempt through `PublicationContext.ExecutionOutput`.
The recorded component must be `bitbucket.ensurePullRequestReviewers`.
Its configuration identifies the same `repository` and `pullRequestId`.
Its `bitbucket.pullRequestReviewersEnsured` output must contain:

```json
{
  "status": "requested",
  "headSha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "requested": [{"uuid": "{11111111-1111-4111-8111-111111111111}", "displayName": "Reviewer"}],
  "alreadyCovered": []
}
```

`already_covered` is also successful. The combined UUIDs must match all
required reviewers. The check never changes the reviewer list.
Issue #9280 owns this producer contract. Its implementation was unavailable
during initial development. Reconcile this interface before merge.
The completion payload and producer identity remain owned by #9279
([draft PR](https://github.com/superplanehq/superplane/pull/9309)).

This change owns `ReviewerIdentity`, `ApprovalEvidence`,
`Client.ListPullRequestReviewActivity`, `Client.ReadPullRequestReviewState`,
and `ReadApprovalEvidence`. Reviewer actions must reuse these operations.
The client returns `approved`, `approval_missing`, `changes_requested`, or
`approval_commit_unverified` for each selected UUID.

## Approval authority

The integration-bound publication store persists evidence under
`bitbucket.review-evidence/<repository UUID>/<PR ID>`. A signed approval event
binds its approval date and reviewer UUID to its full source SHA.
The latest provider approval date, current approved participant, and receipt
must match. Comment dates and event arrival order have no authority.
Conflicting commits for one approval date remain unresolved.

The activity API does not provide historical approval commit attribution.
An existing approval counts when a matching authenticated receipt exists,
including a receipt saved before selection finishes. Otherwise it remains
unresolved. Reconciliation can revoke an approval but cannot create evidence.
A missed unapproval or changes request is detected from current provider state.
A new source SHA invalidates the prior decision and its approvals.

Forge reviewer events contain a reviewer snapshot. The adapter only accepts
evidence for the acting reviewer when its approval timestamp equals the event
timestamp and the source hash is complete. Other snapshots only wake refresh.
Short hashes and unmatched timestamps remain unresolved. Live validation of
these provider fields is required before hosted check activation.

## Status and delivery

Pending selection, missing evidence, and new heads use `INPROGRESS`.
All required approvals, explicit no-review decisions, and release use
`SUCCESSFUL`. Changes requests, selection errors, and lookup errors use
`FAILED`, with separate internal reasons. Status writes include the source
branch `refname` and run URL.

Managed state uses `bitbucket.review-check/<repository UUID>/<PR ID>`.
The provider prefix and integration-bound store isolate identities.
Source and attempt order come from existing trusted publication records.
Findings can use `core.AcceptOrderedWrite` independently with their own key,
including when checks are disabled.

Saved activation settings fence admitted attempts. An off-to-on change must
increase `activationGeneration`. Disable and removal retain a durable fence
and release the check. Provider writes use committed leases, revision and
generation checks, and a deadline anchored to the lease claim. Delivery is
at least once. Recovery can repeat a status after an interrupted write.
Failures stop after ten attempts and remain visible for an explicit retry.

## Connections and rollout

The [build-status API](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commit-statuses/)
documents `read:repository:bitbucket` for Forge and API tokens, including status
creation. The [pull request API](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/)
requires `read:pullrequest:bitbucket` for PR and activity reads.
Workspace and repository access tokens use `repository` and `pullrequest`.
Manual connections also need `webhook` permission and permission to manage
repository hooks. API tokens use `read:webhook:bitbucket` and
`write:webhook:bitbucket` for create/update, plus `delete:webhook:bitbucket`
for cleanup. See the [hook endpoints](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/).
Setup must surface access errors.
Repository tokens remain confined to their configured repository.

[Repository events](https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/)
cover approval, unapproval, changes-request creation/removal, lifecycle updates,
closure, and source pushes. Reopening is reconciled through PR updates and
periodic refresh. The [Forge event reference](https://developer.atlassian.com/platform/forge/events-reference/bitbucket/)
provides `avi:bitbucket:updated:pullrequest-reviewer-status` and
`avi:bitbucket:push:repository`. The prepared manifest adds both subscriptions
and retains existing repository and PR scopes. It has not been deployed.

API processes accept authenticated Forge events. Existing integration-request,
node-request, app-message, and webhook-provisioner workers deliver and monitor
checks. They require database access, encrypted integration credentials, and
`ENCRYPTION_MASTER_KEYS`. Workers require `START_INTEGRATION_REQUEST_WORKER`,
`START_NODE_REQUEST_WORKER`, `START_APP_MESSAGE_WORKER`, and
`START_WEBHOOK_PROVISIONER`. Runners require no change.

Hosted API processes also require `SUPERPLANE_BITBUCKET_FORGE_APP_ID` and
`SUPERPLANE_BITBUCKET_FORGE_INSTALL_URL`. `SUPERPLANE_BITBUCKET_FORGE_JWKS_URL`
can override the default key endpoint. Workers obtain saved Forge system tokens
through the existing encrypted token store. No new configuration key or
migration is required.

Local Compose and the repository chart contain the existing worker switches
and encryption configuration. Hosted chart and launchpad reads through `gh`
returned HTTP 404. Their worker settings, Secret references, Forge settings,
and deployed values remain unverified. Verify them before activation.

Deploy API and worker component support, publish the prepared Forge manifest,
and verify event delivery and credential capabilities before enabling checks.
No deployment occurred during this implementation.

Phase 4 staging verification, shared-interface validation, and combined Phase
5A acceptance remain merge gates. Rebase after companion changes merge.
Current main does not include the late GitHub worker correction described in
[this review response](https://github.com/superplanehq/superplane/pull/9246#issuecomment-6864792706).
That absence blocks Phase 4 sign-off and merge. This change does not repair it
or start Phase 5B.
