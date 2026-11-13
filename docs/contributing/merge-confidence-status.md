# Merge confidence assessment check

The GitHub Merge confidence canvas publishes the check run
`superplane/merge-confidence`. It uses **Set Pull Request Check Run** before
assessment and after report publication. The check shows GitHub's running
spinner throughout assessment and report publication. Runner queue state has
no separate check state.

Scores remain advisory. Critical findings do not fail a complete assessment.
Review request continues to publish a commit status. It stays pending until
all required reviewers approve the current commit. Requested changes fail
that status. An explicit no-review decision succeeds.

| Assessment outcome | Component state | GitHub status | Conclusion |
| --- | --- | --- | --- |
| Active assessment | `pending` | `in_progress` | None |
| Complete assessment and published or unchanged report | `success` | `completed` | `success` |
| Failed or incomplete report, or skipped publication | `failure` | `completed` | `failure` |
| Attempt ends without a final result, including cancellation | `error` | `completed` | `failure` |
| Canvas removal, closed PR, changed commit, or superseded attempt | `error` | `completed` | `failure` |

The component shares ordered publication records, delivery leases, and
integration requests with commit-status components. Existing components keep
their commit-status behavior. The new component uses the same configuration
inputs and state vocabulary. The check output includes the reason and a
SuperPlane run link.

Each attempt owns one check run. The existing publication payload stores its
GitHub ID. A stable external ID identifies the publication and attempt.
Delivery finds that external ID before it repeats an interrupted creation.
Recovery includes all check runs and all result pages for the assessed SHA.
The final result updates the same check run.

Persisted source-run and root-event identities determine order. A delayed
start cannot reopen a terminal result. A newer attempt retires a running
check before publishing its own check, including retries on the same commit.
An older result cannot move to the current head. Source history cleanup does
not delete publication records. Canvas removal resolves pending delivery
through the generic owner check. Deleted organizations cannot publish.

Missing integrations and permanent permission or resource failures stop
polling and record an error. Correct access and retry the publication step.
Transient delivery failures use the existing integration request worker.
If access is revoked, SuperPlane cannot complete an existing GitHub spinner
until access is restored.

## Permissions and deployment

Both generated GitHub App manifests request **Checks: Read and write**.
Changing a manifest does not update an existing hosted App or installation.
The App owner must update the hosted App's permissions. Installation owners
must accept the new access before the template is activated. See
[GitHub check-run permissions](https://docs.github.com/en/rest/checks/runs#create-a-check-run).

API and workers use existing database and encrypted integration credentials,
including `ENCRYPTION_MASTER_KEYS` and existing `SUPERPLANE_GITHUB_APP_*`
configuration where applicable. Workers require
`START_INTEGRATION_REQUEST_WORKER=yes` and `START_NODE_REQUEST_WORKER=yes`.
Runners need no new configuration. This change adds no service, environment
variable, Secret key, protobuf change, or database migration.

Local Compose and the repository Helm chart already configure these workers.
The connected GitHub account cannot read the separate hosted chart or
Launchpad repositories. Their published versions, live worker configuration,
App permissions, and installation acceptance remain unverified.

## Rollout

1. Deploy component support to API and worker processes first.
2. Update the hosted GitHub App to Checks write and obtain installation acceptance.
3. Confirm the test installation has Checks write before starting an assessment.
4. Check branch protection and rulesets for `superplane/merge-confidence`.
5. Refresh only Merge confidence (`risk-score`) through the existing admin reset path.
6. Check reset failures before retrying affected organizations.
7. On a connected test PR, confirm the native spinner starts with assessment.
8. Confirm it continues through report publication and completes on the same check run.
9. Confirm Review request stays pending until all required current-commit approvals exist.

The installation reset action takes one automation template ID. Use
`ResetBuiltInAutomationInAllOrganizations` with `risk-score`; do not reset
all built-in automations. The reset preserves supported check instructions,
enabled checks, model settings, trigger settings, and publication settings.
Existing attempts use their admitted canvas version and saved component.
They finish through the old commit-status path.

During transition, an old status and a new check run can share the same name.
GitHub can require both when that name is a required status check. Verify the
actual branch rules and both results before rollout. Never publish success
for an old status solely to clear a branch requirement. See
[GitHub required status checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches#require-status-checks-before-merging).

Hosted App permission changes, installation acceptance, deployment, reset,
branch-rule verification, and connected-PR checks must be verified in the
rollout environment. Local suites do not establish these live conditions.
