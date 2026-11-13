# Bitbucket reviewer actions

Use these actions on ordinary canvases. This change does not enable the built-in Review request switch.

## Credentials

Use an Atlassian API token with these scopes:

| Operation | Required API token scopes |
| --- | --- |
| List candidates and repository reviewers | `read:repository:bitbucket`, `read:pullrequest:bitbucket`, `read:user:bitbucket` |
| Ensure reviewers | All read scopes above and `write:pullrequest:bitbucket` |

The token account must also have **admin permission on the repository**.
Repository read scope alone does not grant admin access.
The effective-permissions endpoint returns the highest direct or inherited group permission for each user.
Permission failures remain errors. SuperPlane does not treat them as empty discovery.

Hosted Forge connections currently lack `read:user:bitbucket` in the app manifest.
Workspace and repository access tokens do not provide the account verification capability required here.
The new actions and reviewer picker reject these connection modes before setup or reviewer mutation.
Existing Bitbucket actions keep their credential support.
Do not add scopes to an installation or substitute another credential automatically.

The manual API token transport and unsupported-mode errors have automated coverage.
Live API token and Forge validation require a staging repository and authorized credentials.
No live provider validation is claimed by this change.
Verify account status responses, inherited permissions, reviewer updates, and retained approvals in staging before merge.
A future Forge capability change requires provider verification and an explicit installation upgrade before setup can allow Forge.

Sources: [effective repository permissions](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-workspaces/#api-workspaces-workspace-permissions-repositories-repo-slug-get),
[user profiles](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-users/#api-users-selected-user-get),
and [pull requests](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/).

## Discovery contract

`bitbucket.listPullRequestReviewCandidates` accepts `repository` and `pullNumber`.
The default channel emits `bitbucket.pullRequestReviewCandidates` with these fields:

| Field | Meaning |
| --- | --- |
| `provider`, `integrationId`, `repositoryId`, `pullNumber` | The identity boundary. `provider` is `bitbucket`. The repository ID is a UUID. |
| `repository` | The current workspace/repository path. It is a label and API address. |
| `headSha`, `state`, `author` | The PR snapshot used for discovery. |
| `candidates` | Eligible human identities with relevance evidence. |
| `contributorCommits`, `contributorHistoryLimited` | The size and limit of the recent commit sample. |

Identities contain canonical unbraced `uuid` and a separate `displayName`.
Display names and nicknames never identify accounts.
The repository-scoped `reviewer` resource uses UUID as its value and display name as its label.
It returns eligible fallback users. The ensure action rejects the PR author again at execution.

Discovery reads every effective-permission page, then verifies each account through its profile.
Only active `user` accounts with effective `write` or `admin` access qualify.
App users, known bot nicknames, inactive accounts, and the PR author do not qualify.
Missing account status or malformed identity data remain errors.
Bitbucket cannot distinguish an undisclosed service account that uses a normal human account.

Evidence contains `default_reviewer` for effective repository or project defaults.
It contains `recent_contributor` for linked authors in the latest 100 repository commits.
The commit sample follows provider pagination until it reaches 100 commits or history ends.
Unlinked commit authors do not establish identity.
Empty evidence identifies an eligible fallback candidate.
These signals do not claim file ownership. SuperPlane does not apply GitHub CODEOWNERS semantics to Bitbucket.
Discovery rechecks the PR head, state, and author before it emits results.
Empty successful discovery emits an empty candidate array.

## Reviewer request contract

`bitbucket.ensurePullRequestReviewers` accepts `repository`, `pullNumber`, `expectedHeadSha`, and `reviewers`.
The assessed SHA is required. Each selected reviewer supplies a `uuid`; `displayName` is optional.
The default channel emits `bitbucket.pullRequestReviewersEnsured`.
It includes the same identity boundary, `executionId`, `headSha`, and these fields:

| Field | Meaning |
| --- | --- |
| `status` | `requested`, `already_covered`, or `stale_pr`. |
| `reason` | `closed` or `head_mismatch` for stale PRs. |
| `requested` | Selected identities that required a reviewer update. |
| `alreadyCovered` | Selected identities already assigned as reviewers. |
| `unresolvedApprovals` | Participant approvals whose assessed commit is unverified. |

Errors fail the execution. They do not emit a success or empty candidate result.
A closed or changed PR before mutation produces `stale_pr` without a write.
A changed PR after mutation remains an error because the update may already exist.

`Client.ListEligibleReviewers`, `Client.ListPullRequestReviewCandidates`, and `Client.GetPullRequestReviewSnapshot` are shared client operations.
`Client.EnsurePullRequestReviewersSerialized` is the shared reviewer write boundary.
It takes a `ReviewerRequest`, a stable repository UUID, and the integration-scoped publication store.
`ReviewerWriteKey` adds provider, repository UUID, and PR number to that store boundary.
All automation reviewer writers must use this method and key.

The method reuses the durable publication lease. A busy lease returns an error before provider mutation.
HTTP requests have a deadline 15 seconds before the two-minute lease expires.
There is no database transaction open during provider calls.
A worker interruption leaves the lease for bounded expiry; a retry reads current provider state again.

Each attempt rechecks effective eligibility and the latest PR snapshot immediately before a write.
The PUT body contains only `reviewers`, merged with current and previously observed reviewer UUIDs.
An available strong ETag is sent as `If-Match`. Bitbucket does not document conditional PUT enforcement.
A fresh GET verifies all intended and preserved reviewers and existing approval flags after the update.
Conflicts and missing reviewers allow at most three attempts.
Each attempt reads eligibility and current PR state again.
Provider errors, lost approvals, verification failures, and exhausted reconciliation remain visible errors.

This method does not promise atomic additive updates.
External edits can race between reads and writes. Edits never visible in a snapshot cannot be recovered.
Reconciliation can restore a reviewer that another user concurrently removed.
Use the final provider state to investigate uncertain outcomes.

## Companion and merge requirements

Issue #9280 owns candidate discovery, reviewer-request outputs, and reviewer client operations described here.
Issue #9279 owns immutable completion payloads and producer identity. See PR #9309.
Issue #9281 owns durable approval-to-commit evidence and review-state operations. No companion PR was available during implementation.

A participant's `approved: true` does not establish approval of the assessed SHA.
This implementation preserves observed approval flags but reports them as unresolved.
It counts an existing reviewer assignment as covered, without claiming that its approval applies to this commit.
Integrate the authoritative approval-evidence operation from #9281 before merge.
Do not accept caller-supplied approval evidence as authority.
Rebase after companion changes merge and run combined Phase 5A validation before Phase 5B starts.

Main at `2ee58506a863edcdba6120168383994e81fc8075` lacks fix `72b4cdf1741e924240008427237e247d06395416`.
The affected `retainDeliveredCheckRun` path also lacks the equivalent terminal-result restoration.
This remains a Phase 4 sign-off and Phase 5A merge blocker. This change does not repair GitHub delivery.
Phase 4 staging validation, live credential validation, and shared-interface validation remain merge gates.

The API and existing node execution workers use existing integration credentials, database access, and publication storage.
This change adds no worker, service endpoint, environment variable, Secret key, or deployment configuration.
It does not change the Forge manifest or activate a built-in automation.
