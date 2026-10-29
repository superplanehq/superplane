# Factory VCS providers

A factory workspace has one Git host. That host is the version control
provider stored on factory onboarding. GitHub and Bitbucket are the hosts
that work today. This document says what a new host must implement, where
those pieces live, and how GitLab should join them.

A workflow integration is a different layer. It lets a canvas call one
product. `pkg/integrations/gitlab/` is that layer for GitLab. It does not
make GitLab a factory host. The host screen still shows GitLab as
unavailable.

Use [.cursor/skills/superplane-vcs-provider/SKILL.md](../../.cursor/skills/superplane-vcs-provider/SKILL.md)
to plan a host. Confirm the plan with a person before you write code.

Dates below are the merge day in Europe/Belgrade.

## What the two hosts already do

`FactoryVCSCapabilitiesFor` in `pkg/models/factory_onboarding.go` is the
product list.

| Capability | GitHub | Bitbucket |
| --- | --- | --- |
| Pull request closure | yes | yes |
| Pull request feedback | yes | yes |
| Pull request checks | yes | yes |
| Merge confidence | yes | yes |
| Velocity | yes | yes |
| Board merge | yes | yes |
| Board close | yes | yes |
| Issue intake | yes | no |

An empty `vcs_provider` means GitHub. Existing workspaces depend on that.

GitHub also has issue intake, visual evidence, and a scanned review-bot
list. Bitbucket has scheduled intake and the other rows in the table.
Bitbucket has no issue intake and no visual-evidence template. A typed bot
name field replaces the scanned list.

## Lessons from the commits

The lists below are the changes that should be in the plan for the next
host. They are grouped by what landed, what was repaired, and what was
removed. Squash merges on `main` hide the intermediate commits. The text
comes from the pull requests.

### GitHub

The factory host started as a public GitHub App, then became a provider
catalog, then grew operational fixes.

| Date | Change | What it established |
| --- | --- | --- |
| 2026-08-27 | [#6885](https://github.com/superplanehq/superplane/pull/6885) Install the public GitHub App | Cloud uses one public App. The process holds the App id, slug, private key, webhook secret, and OAuth client. Self-host kept the private-app manifest. The same installation can link to more than one SuperPlane organization. |
| 2026-09-05 | [#7215](https://github.com/superplanehq/superplane/pull/7215) Keep setup on onboarding | An owner-approve email can omit CSRF state. The return path is stored on the integration. With factories on, the callback returns to onboarding. |
| 2026-09-11 | [#7371](https://github.com/superplanehq/superplane/pull/7371) Redirect update callbacks | An update callback can omit state. Redirect recognized shapes. Do not trust the installation id in that URL. |
| 2026-09-27 | [#7853](https://github.com/superplanehq/superplane/pull/7853) Remove App user authorization | This replaced the broad user grant with the linked identity. |
| 2026-09-28 | [#7878](https://github.com/superplanehq/superplane/pull/7878) Revert that connection change | [#7853](https://github.com/superplanehq/superplane/pull/7853) and two follow-ups left onboarding broken. The tree was restored. Do not remove a working grant until the replacement is proven. |
| 2026-09-29 | [#7910](https://github.com/superplanehq/superplane/pull/7910) Simplify onboarding | Replace per-integration App setup with `vcs_provider_installations` and related catalog tables. Onboarding lists repositories across installations. Members can link more than one GitHub identity. |
| 2026-09-30 | [#7966](https://github.com/superplanehq/superplane/pull/7966) Speed up repository sync | Target one installation. Bound retries. Isolate failures. Interactive sync has priority. A repository appears after push access is verified. |
| 2026-09-30 | [#7970](https://github.com/superplanehq/superplane/pull/7970) Show approval only for a real request | A direct install from the GitHub listing also omits setup state. Show the approval page only when a pending request matches. |
| 2026-10-01 | [#7983](https://github.com/superplanehq/superplane/pull/7983) Register repository hooks | The App webhook was not a substitute for a repository hook. The production App subscribed to no repository events, so triggers never ran. Repository hooks deliver node events. The App endpoint keeps installation, installation-repository, and member events. Both paths together run a trigger twice. |
| 2026-10-03 | [#8097](https://github.com/superplanehq/superplane/pull/8097) Improve onboarding steps | Connect is required even when sign-in already has a GitHub identity. Add the organization step, account picker, skeleton rows, and polling for install requests. GitHub sends no webhook for those requests. |
| 2026-10-03 | [#8105](https://github.com/superplanehq/superplane/pull/8105) Remove uninstalled organizations sooner | GitHub often omits the installation `deleted` webhook. While the chooser is open, verify installations and delete a 404. The five-minute reconciliation stays as the fallback. |
| 2026-10-08 | [#8308](https://github.com/superplanehq/superplane/pull/8308) Create the App in workspace setup | Self-host can create the public App from Connect and store it on the installation. Cloud keeps environment credentials. |
| 2026-10-10 | [#8358](https://github.com/superplanehq/superplane/pull/8358) Drop the private App login client | The factory App does not need a user login client. Sign-in stays on `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET`. |
| 2026-10-14 | [#8477](https://github.com/superplanehq/superplane/pull/8477) Make the self-hosted App public | A private App can be installed only on the account that created it, so organization repositories never appear. Organization integration Apps stay private. |
| 2026-10-16 | [#8607](https://github.com/superplanehq/superplane/pull/8607) Open organization choice after install | After install, unfinished workspace setup opens Choose organization. Two unfinished workspaces return home. |
| 2026-10-19 | [#8676](https://github.com/superplanehq/superplane/pull/8676) Read-only token for context repositories | The workspace token includes only the workspace repository. Context clones need a separate read-only token. A repository access change schedules a full catalog refresh. |

### Bitbucket

Bitbucket Cloud became a factory host in slices. The first slice was
connect and Implement. Parity with GitHub arrived in later pull requests.
Each later slice left a gap that another pull request closed.

| Date | Change | What it established |
| --- | --- | --- |
| 2026-02-17 | [#3039](https://github.com/superplanehq/superplane/pull/3039) Add the Bitbucket integration | Workflow integration only. Workspace auth, repository list, and an On Push trigger. This is not the factory host. |
| 2026-10-06 | [#8165](https://github.com/superplanehq/superplane/pull/8165) Support Bitbucket factory workspaces | Feature flag `factory_bitbucket`. Choose Bitbucket, select a repository, and open a pull request from Implement. Merge, close, mergeability, and Velocity return not supported. No review feedback, pipeline checks, or issue intake. GitLab stays unavailable. |
| 2026-10-07 | [#8233](https://github.com/superplanehq/superplane/pull/8233) Connect accounts with Forge | A workspace admin installs the Forge app. SuperPlane uses the bot token for repositories and git push. Account link is not a sign-in method. When Forge is unset, the token screen remains. |
| 2026-10-08 | [#8325](https://github.com/superplanehq/superplane/pull/8325) Onboarding and discovery | An installed workspace with no writable repositories must not look like "install again". Discovery needs a Forge delivery. Bitbucket OAuth does not list installations. Failed deliveries and stale installations have metrics. |
| 2026-10-09 | [#8327](https://github.com/superplanehq/superplane/pull/8327) Checks and Velocity | Build checks, feedback, closure, merge confidence, and board merge. Velocity records stay separate by provider. Authors match on account UUID. Repository access tokens work. |
| 2026-10-12 | [#8363](https://github.com/superplanehq/superplane/pull/8363) Forge pull request and build events | A new endpoint receives those events, routes them to webhooks, and resumes the build wait. |
| 2026-10-12 | [#8384](https://github.com/superplanehq/superplane/pull/8384) Exclude bots from Velocity | Exclude Forge apps by account type. Keep recognized SuperPlane agent merges as automated output. A name check is not enough. Refresh stored Velocity data after deploy. |
| 2026-10-13 | [#8337](https://github.com/superplanehq/superplane/pull/8337) Forge `uuid` | A side app needs its own dependency patches. |
| 2026-10-14 | [#8338](https://github.com/superplanehq/superplane/pull/8338) Forge `prosemirror-view` | Same. An npm override was required because the parent package did not move. |
| 2026-10-14 | [#8484](https://github.com/superplanehq/superplane/pull/8484) Remove a linked account | Connect accepted Bitbucket and remove accepted only GitHub. Both sides now use `LinkableProviders`. |
| 2026-10-15 | [#8509](https://github.com/superplanehq/superplane/pull/8509) Verify hand-off | Accept an open pull request URL only for the workspace host. The Bitbucket response must include the title, the HTML link, and the source repository. |
| 2026-10-15 | [#8524](https://github.com/superplanehq/superplane/pull/8524) Account switcher | The GitHub signed-in line became shared UI. Bitbucket lists linked accounts and can connect another. `prompt=select_account` was added without proof that Bitbucket honors it. The linked-account switcher still has to work. |
| 2026-10-16 | [#8588](https://github.com/superplanehq/superplane/pull/8588) Remove `factory_bitbucket` | The flag comes off only after the host is a normal choice. GitLab stays disabled. Stored feature ids can remain in the database. |
| 2026-10-16 | [#8596](https://github.com/superplanehq/superplane/pull/8596) Bot name field | Bitbucket has no review-bot catalog. Show a name field. Do not show an empty scanned list. |
| 2026-10-17 | [#8623](https://github.com/superplanehq/superplane/pull/8623) Task link in the reply | The acknowledgement comment links the SuperPlane task. Put that link in the first feedback template. |
| 2026-10-18 | [#8625](https://github.com/superplanehq/superplane/pull/8625) Rich check rows | Waiting, passed, and failed runs use the GitHub row: icon, linked name, description, and badge. New factories use the same check titles. |
| 2026-10-25 | [#8660](https://github.com/superplanehq/superplane/pull/8660) Install spinner across refresh | The attempt lives in `sessionStorage` per organization. The five-minute window does not restart on reload. A new tab does not see it. |
| 2026-10-25 | [#8886](https://github.com/superplanehq/superplane/pull/8886) Patch `fast-uri` | Another Forge lockfile fix. |

### Rules that follow from those changes

- Plan every capability before the first slice. Ship them in order. Turn a
  capability on in the same slice as the operation.
- Keep a feature flag until connect, Implement, checks, feedback, merge,
  Velocity, and Verify hand-off all work.
- Put the account switcher and account removal in the first connect slice.
- Keep manual connect available when the hosted app is unset.
- Account link and sign-in are separate. Add a provider to
  `LinkableProviders` once.
- Do not deliver one repository event through two channels.
- Poll or verify when the host sends no delete or request webhook.
- A callback with no state must return the browser to a safe page.
- Exclude Velocity bots by account type. Refresh stored rows after you
  change that rule.
- If the write token is limited to one repository, mint a read token for
  context repositories.
- If the host has no bot catalog, use a typed name field.
- The feedback reply includes the task link.
- Check rows match the GitHub console row.
- An in-progress install survives a refresh.
- A side app, such as `forge/bitbucket/`, owns its dependency patches.
- An unknown provider must not run the GitHub client. `Select` in
  `pkg/factories/vcs/provider.go` still returns GitHub for every name except
  Bitbucket. `openFactoryVCS` special-cases Bitbucket first. Add the new
  host there.

## Where a new host is implemented

Start from the Bitbucket call sites when the new host has its own client.
Start from the GitHub catalog only when the host has a real installation
object, an installation webhook, and an installation token. GitLab has none
of those. Use the GitLab section below.

### Identity and validation

| Concern | Where |
| --- | --- |
| Provider id | `ProviderGitHub` and `ProviderBitbucket` in `pkg/models/constants.go` |
| Onboarding validation, empty means GitHub, clone URL, capabilities | `pkg/models/factory_onboarding.go` |
| Pull request provider, URL parser, `owner/name` check | `pkg/models/factory_pull_request.go` |
| Linked accounts | `LinkableProviders` in `pkg/models/account_linked_account.go` |
| Workspace column | `vcs_provider` on the factory. Proto field `vcs_provider` in `protos/factories.proto` |

`validPullRequestRepository` accepts only two path segments. Keep that rule
for GitHub and Bitbucket. A host with subgroup paths needs its own check.

`AppRepositoryID` is the numeric id. `AppRepositoryExternalID` is the
Bitbucket repository UUID. Use the numeric field for a numeric project id.

### Connect and catalog

| Concern | GitHub | Bitbucket |
| --- | --- | --- |
| Installation row | `vcs_provider_installations` in `pkg/models/vcs_provider_catalog.go` | `bitbucket_forge_installations` in `pkg/models/bitbucket_forge_installation.go` |
| Sync | `pkg/githubapp/catalog.go`, `pkg/workers/vcs_provider_catalog_worker.go` | Forge deliveries in `pkg/public/bitbucket_forge.go` |
| Onboarding API | `pkg/grpc/actions/me/vcs_provider_onboarding.go` | `pkg/grpc/actions/me/vcs_provider_onboarding_bitbucket.go` |
| Live uninstall check | `pkg/grpc/actions/me/vcs_provider_installation_checks.go` | Stale Forge token and delivery metrics |
| OAuth | `pkg/authentication/authentication.go` | `pkg/authentication/bitbucket_oauth.go` |
| Side app | Public GitHub App, created from setup for self-host | `forge/bitbucket/` |

`AccountLinkedAccount` stores identity. It stores no token and grants no
session. The factory credential lives on the installation or the
integration.

### Factory operations and canvases

| Concern | Where |
| --- | --- |
| Provider interface | `pkg/factories/vcs/provider.go` |
| Host selection | `openFactoryVCS` in `pkg/grpc/actions/factories/factory_vcs.go` |
| Bitbucket client | `pkg/grpc/actions/factories/factory_bitbucket_client.go` |
| Repository choice | `select_factory_vcs_repository.go`, `select_bitbucket_repository.go`, `update_factory_repository_bitbucket.go` |
| Templates | `factory_app_templates.go` and `templates/github/`, `templates/bitbucket/` |
| Feedback | `create_factory_pr_feedback_handler.go`, `pr_feedback_bitbucket_runner.go` |
| Custom automation payload | `custom_automation.go`, `custom_automation_run.go` |
| Verify hand-off | `hand_off_work_order.go` |
| Issue intake guard | `create_factory_intake.go` rejects GitHub issue sources on a Bitbucket workspace |
| Velocity | `pkg/workers/github_velocity.go`, `pkg/workers/bitbucket_velocity.go`, branch in `pkg/workers/factory_velocity_sync_worker.go` |
| Branch link | `pkg/components/factory/branch_artifact_url.go` |
| Context clone URL | `WorkspaceContextRepoURL` in `pkg/components/runner/workspace_agent_resources.go` |
| Context repo access | `authorizeContextRepository` in `pkg/grpc/actions/factories/factory_agent_resources.go` is GitHub-only today |

Built-in automation reset must keep the new host's settings. Follow
[pkg/grpc/actions/factories/AGENTS.md](../../pkg/grpc/actions/factories/AGENTS.md)
and extend `reset_built_in_automation_settings_test.go`.

### Onboarding UI

| Concern | Where |
| --- | --- |
| Host ids and the GitLab "soon" flag | `onboardingFixtures.ts` |
| Remembered host | `onboardingVcsHostChoice.ts` |
| Host screen | `first-run/FirstRunHostScreen.tsx` |
| Bitbucket flow | `useFirstRunBitbucket.ts`, `first-run/FirstRunBitbucketForgeScreen.tsx` |
| Install attempt | `bitbucketInstallationAttempt.ts` |
| Signed-in line | Shared with the GitHub choose screen after [#8524](https://github.com/superplanehq/superplane/pull/8524) |
| Check rows | Console code that reads the GitHub wait node and the Bitbucket wait node |

`useFirstRunSetupFlow` returns immediately for `gitlab`. The host button
stays disabled. Tests in `FirstRunSetup.bitbucket.spec.tsx` lock that.

### Integration components

Factory templates call integration components. Examples:
`github.createPullRequest`, `bitbucket` pull request components, and the
GitLab components that already exist (`create_merge_request.go`,
`accept_merge_request.go`, `on_merge_request.go`, `on_pipeline.go`,
`on_merge_comment.go`).

Reuse that package. Extend it when a factory call needs a method it does
not have. Do not put a second HTTP client in `pkg/grpc/actions/factories`
when the integration client can do the call. Bitbucket grew
`factory_bitbucket_client.go` because the factory API and the integration
grew together. For GitLab, start from `pkg/integrations/gitlab/client.go`.

Webhook setup for GitLab is `hooks.go` and `webhook_handler.go`. Incoming
checks use `verifyWebhookToken` in `common.go`, which compares
`X-Gitlab-Token`.

### Configuration

Document a hosted app in `.env.example`, `docker-compose.dev.yml`, and
[connecting-to-3rdparty-services-from-development.md](connecting-to-3rdparty-services-from-development.md).

Integration sync, including GitLab token refresh, runs in
`pkg/workers/integration_request_worker.go`. The API process starts the
OAuth redirect. Both processes need the client id and secret if refresh or
the redirect reads the environment. The current GitLab integration stores
the client id and secret on the integration and refreshes from there. The
hosted path should write those env values onto the integration at connect
so the existing refresh keeps working.

Name the variables `SUPERPLANE_GITLAB_OAUTH_CLIENT_ID` and
`SUPERPLANE_GITLAB_OAUTH_CLIENT_SECRET`. That matches
`SUPERPLANE_JIRA_OAUTH_*`. Do not invent a second prefix.

## GitLab

GitLab.com can match the factory product. It cannot copy the GitHub App
catalog or the Bitbucket Forge app. Those two hosts install an application
that acts as a bot. GitLab.com authorizes a user. The first slice is
GitLab.com only. Self-managed GitLab is a later slice on the same code, with
a base URL.

The workflow integration already supports GitLab.com and a custom base URL,
personal access tokens, per-connection OAuth, project hooks, merge requests,
notes, pipelines, and commit statuses. Factory work reuses it.

### Differences that change the design

**There is no installation object.** GitHub receives installation and member
webhooks and stores them in `vcs_provider_installations`. Bitbucket receives
Forge install events and stores them in `bitbucket_forge_installations`.
GitLab.com has a user-owned OAuth application. Each user authorizes it.
There is no install event, no install request, and no installation id.
Do not insert GitLab rows into the GitHub catalog tables.

**The token is the user on GitLab.com Free.** Project access tokens and
group access tokens require Premium or Ultimate on GitLab.com. They exist
on every self-managed license. A factory that must work for a Free group
acts as the authorizing user. Commits, comments, and merges show that user.
Do not block Connect on a bot token. A later slice can store a project
access token when the group can create one. Velocity uses `author.bot` to
skip real bots. It must not skip the authorizing user.

**Tokens expire.** GitHub mints an installation token per call. Bitbucket
refreshes the Forge system token from deliveries. GitLab OAuth access
tokens expire and refresh through `pkg/integrations/gitlab/auth.go`.
Personal access tokens expire too. Factory calls use that client. Show an
expired connection on the connect screen.

**The project path is not `owner/name`.** GitLab `path_with_namespace` can
be `group/subgroup/project`. The pull request number is the merge request
IID. The stable id is the numeric project id. Store the path as the
repository string and the project id in `AppRepositoryID`. The URL shape is
`https://gitlab.com/<path>/-/merge_requests/<iid>`. The branch URL uses
`/-/tree/<branch>`. The clone URL is `https://gitlab.com/<path>.git`.
Parse these in `parsePullRequestURL` and `branch_artifact_url.go`. Leave
the GitHub and Bitbucket patterns on two segments.

**Webhooks are per project.** Project hooks are on every tier. The creator
needs the Maintainer role. Group hooks are Premium or Ultimate, and a group
hook plus a project hook delivers the same event twice. Use project hooks
only. Create them through the existing hooks client.

GitLab 19.1 sends a signing token. The header is `webhook-signature` over
`{webhook-id}.{webhook-timestamp}.{body}` (Standard Webhooks). The current
handler checks only `X-Gitlab-Token`. New factory hooks set `signing_token`
and the shared handler verifies that signature. Keep the plain-token check
for hooks that already exist. One hook per project. Subscribe to merge
request, note, pipeline, and push events for the factory templates.

**The catalog is a list.** While the chooser is open, list groups the user
can access, then projects with `include_subgroups=true`. Include a personal
namespace row. Show projects where the user is at least Developer. Hook
creation needs Maintainer. If GitLab returns 403, say that the Maintainer
role is required. A 401 means the grant expired. There is no uninstall
webhook.

**Merge is not the GitHub method enum.** The GitLab merge API is
`PUT /projects/:id/merge_requests/:iid/merge` with `sha`, `squash`, and
optional `auto_merge`. The project setting `merge_method` is merge commit,
semi-linear, or fast-forward. Squash is a separate project option: never,
always, default on, or default off. There is no rebase argument on the
call. Allowed board methods come from those project settings.

Read `detailed_merge_status`, not the older `merge_status`. Map
`mergeable` to can-merge. Map `checking`, `preparing`, `ci_still_running`,
and `approvals_syncing` to retry. Map `not_approved`,
`discussions_not_resolved`, `draft_status`, `conflict`, `need_rebase`,
`ci_must_pass`, `requested_changes`, and `commits_status` to the existing
blocked-reason names. Add a name only when none fits.

**Checks are pipelines.** Wait on the pipeline for the merge request head
SHA. Also read commit statuses so external CI appears. Do not render one
row per job. The console row uses the pipeline name, status, and web URL,
with the same titles as GitHub (`Waiting for checks on`, `Checks passed on`,
`Fixing failed checks on`, `Failed checks`).

**Reviews are notes.** Feedback uses the note hook and the merge request
actions `approved` and `unapproved`. The acknowledgement is a note with the
task link, as on Bitbucket. Award emoji can wait. There is no bot catalog.
Use the typed bot-name field. Skip a note whose author has `bot: true` when
the name matches, and skip the acting user so a reply does not loop.

**The clone token is not scoped per repository.** Git credentials use
`oauth2` as the username and the access token as the password. That token
can push every project the user can push. Context repository clone uses the
same token. On save, list the project with the token and reject a project
the token cannot see. Do not copy the GitHub read-only installation token
until a project access token exists for that repository.

**Account link is not sign-in.** Add `gitlab` to `LinkableProviders` so
connect and remove stay together. `DELETE` of a GitLab linked account is
rejected today. The switcher lists linked GitLab accounts and can start
another authorize. Do not depend on an account-picker query parameter until
the current GitLab authorize docs name it. Store each grant as its own
GitLab integration, because the token belongs to that user. The workspace
`vcs_integration_id` selects the active grant. The linked-account row still
stores no token.

**Issue intake stays off in the first slices.** `gitlab.onIssue` already
exists. Bitbucket still has issue intake off. Turn it on only when a person
asks. The onboarding issues step must not offer GitLab issues before that.

**Self-managed is the same product with a base URL.** One
`SUPERPLANE_GITLAB_OAUTH_*` pair on gitlab.com does not authorize a
self-managed instance. That instance needs its own OAuth application, and
its network must reach SuperPlane for webhooks. Store `base_url` on the
workspace from the first slice. Empty means `https://gitlab.com`. The first
slice hard-codes that host in the URL parsers, the way Bitbucket
hard-codes `bitbucket.org`. The later slice reads the stored base URL.
Personal access token plus base URL is the manual path for self-managed,
which the integration already supports.

### GitLab slice order

1. **Connect and Implement.** Flag `factory_gitlab`. Hosted OAuth when both
   env vars are set. Personal access token when they are empty. Account
   switcher and remove. Group and project list. Save path and project id.
   Line template calls `gitlab.createMergeRequest`. Clone URL uses the user
   token. Capabilities on: none of the board actions yet.
2. **Pull request lifecycle.** Close, merge, mergeability, branch links, and
   Verify hand-off. Turn on closure, board merge, board close, and merge
   confidence.
3. **Events.** Project hooks, signing-token verification, feedback, check
   wait, rich check rows, task link, typed bot names. Turn on feedback and
   checks.
4. **Velocity.** List merged merge requests. Author id is the GitLab user
   id. Skip `author.bot`. Records stay on provider `gitlab`. Turn on
   Velocity.
5. **Remove the flag** after those four slices work on GitLab.com.

Later, only when a person asks:

- Issue intake through `gitlab.onIssue`.
- Visual evidence.
- Self-managed base URL in onboarding.
- A project access token bot for groups that can create one.

### GitLab file checklist

Reuse:

- `pkg/integrations/gitlab/` client, auth, hooks, merge request actions, and
  triggers.
- `web_src/src/assets/icons/integrations/gitlab.svg`.
- `docs/components/GitLab.mdx` for the workflow components. Extend it when a
  component gains a factory behavior.

Extend:

- `pkg/models/constants.go`, `factory_onboarding.go`, `factory_pull_request.go`,
  `account_linked_account.go`.
- `openFactoryVCS` and a `gitlabProvider` beside `bitbucketProvider`.
- `FactoryVCSCapabilitiesFor`.
- `pkg/authentication/authentication.go` for connect-only OAuth. Reject
  GitLab as a sign-in method, as Bitbucket does.
- `pkg/features/features.go` and `web_src/src/lib/experimentalFeatures.ts`
  for `factory_gitlab`.
- Onboarding files listed above. Remove the early return for `gitlab` when
  the flag is on. Keep the button disabled when the flag is off.
- `templates/gitlab/` copied from `templates/bitbucket/` and retargeted to
  `gitlab.*` components.
- `factory_app_templates.go` embed and the reset test.
- `hand_off_work_order.go`, `branch_artifact_url.go`,
  `WorkspaceContextRepoURL`, `create_factory_intake.go`.
- `pkg/workers/factory_velocity_sync_worker.go` with a GitLab reader.
- `pkg/integrations/gitlab/common.go` for signing-token verification.
- `.env.example`, `docker-compose.dev.yml`, and the third-party connect doc.

Skip:

- `pkg/githubapp/` and `vcs_provider_installations` for GitLab rows.
- `forge/bitbucket/` as a pattern. GitLab has no Forge app.
- A second GitLab HTTP client.
- Group webhooks.
- Issue intake and visual evidence in slices 1–4.
