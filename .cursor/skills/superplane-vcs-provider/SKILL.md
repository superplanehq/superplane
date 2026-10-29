---
name: superplane-vcs-provider
description: >-
  Plan a new SuperPlane factory Git host before writing code. Use when adding
  a VCS provider such as GitLab, or when changing factory onboarding, pull
  request automation, or Velocity for GitHub or Bitbucket. Read the VCS
  provider specification, compare it with the current code, and confirm the
  plan with the human before implementation.
---

# Plan a factory VCS provider

Use this skill before you add or extend a factory Git host. GitHub and
Bitbucket are the finished hosts. A workflow integration under
`pkg/integrations/<name>/` is not a factory host. GitLab already has that
integration. It is not a factory host yet.

The specification is
[docs/contributing/vcs-provider.md](../../docs/contributing/vcs-provider.md).
Read it in full before you ask the human anything the document already
answers. If the code and the specification disagree, trust the code for
behavior, then update the specification in the same change.

Do not write provider code until the human confirms the plan at the end of
this skill. If they already named the host, start at **Read the host**. If
they did not, ask one question and stop:

> Which Git host do you want to plan? For example, GitLab.

## Read the host

Search the repo before you ask the human anything the code already answers.

- Provider ids: `pkg/models/constants.go` and
  `ValidateFactoryOnboardingVCSProvider` in `pkg/models/factory_onboarding.go`.
- Capabilities: `FactoryVCSCapabilitiesFor` in the same file.
- The operation seam: `openFactoryVCS` in
  `pkg/grpc/actions/factories/factory_vcs.go`. An unknown provider must not
  fall through to GitHub. `pkg/factories/vcs/provider.go` still does that for
  every name except Bitbucket. Do not trust its package comment.
- Templates: `pkg/grpc/actions/factories/factory_app_templates.go` and
  `pkg/grpc/actions/factories/templates/<provider>/`.
- Onboarding UI: `web_src/src/pages/factories/pages/onboarding/`.
  `VcsHostId` already includes `gitlab`, and the host button stays disabled.

Tell the human what already exists, in two or three sentences. Then walk
the steps below. Skip a question when the specification or the code already
answers it, and say the fact you will use.

When the host is GitLab, use the GitLab section of the specification as the
plan. Present that plan. Ask the human to confirm it or to change one
decision. Do not open a second research pass unless they reject a decision.

## How to ask

- One step per turn when a decision is still open. Wait for the answer.
- Ask only what this host still needs. Do not repeat the whole list.
- Propose the default from the specification. Ask the human to accept it
  or change it.
- If the human already gave a full spec, map it onto the steps, list the
  gaps, and ask only for the gaps.
- User-facing names and helper text follow
  [ui-copy](../../.agents/skills/ui-copy/SKILL.md) and
  [simplified-technical-english](../../.agents/skills/simplified-technical-english/SKILL.md).
  The product name is SuperPlane.

## Step 1. Name the product result

State the factory behaviors this host will match. Use
`FactoryVCSCapabilities` names:

- Implement opens a pull request (a GitLab merge request is that object).
- Close, merge, and mergeability on the board.
- Pull request feedback, checks, and merge confidence.
- Velocity.
- Verify hand-off from a pull request URL.
- Context repository clone.
- Issue intake, only when the human asks. Bitbucket still has it off.
  GitHub has it on.

Say which of these stay off in the first slice, and which slice turns each
one on. A capability flag turns on only when the operation and the template
exist. Bitbucket's first change returned a not-supported error for merge,
close, mergeability, and Velocity. Later changes filled those gaps one by
one and left follow-up bugs. Name the slices before you write code.

## Step 2. Choose the connect model

Read the history section of the specification. Pick one model and say why.

- **App installation.** GitHub App or Bitbucket Forge. The app has a bot
  token. The linked account is identity only and stores no token.
- **User grant.** The token is the user. GitLab.com OAuth and a personal
  access token are this model. Say that commits and comments show that user.
- **Both.** Hosted connect when the deployment has the public app. Manual
  connect when those variables are empty. Bitbucket keeps the token screen
  when Forge is unset. Keep that rule.

Also decide:

- Is account link a sign-in method? Bitbucket link is not. GitHub sign-in is
  a separate OAuth client.
- Connect and remove use `models.LinkableProviders`. Add the provider to
  that list once. A remove route that accepts only GitHub was a later
  Bitbucket fix.
- The account switcher is part of the first onboarding slice. Bitbucket
  added it later. The switcher of already linked accounts must work even
  when the authorize URL ignores an account-picker parameter.
- An install that is in progress must survive a page refresh. Bitbucket
  stores that attempt in `sessionStorage`. A synchronous OAuth return does
  not need that spinner.

## Step 3. Choose repository identity

Answer these before you touch validation:

- What is the stable id? GitHub uses a numeric repository id. Bitbucket uses
  a repository UUID in `AppRepositoryExternalID`. GitLab uses the numeric
  project id in `AppRepositoryID`.
- What string does the workspace store? GitHub and Bitbucket use
  `owner/name`. GitLab uses `path_with_namespace`, which can contain more
  than one slash.
- Which URL shapes must parse? Include the pull request URL, the branch URL,
  and the clone URL. Add the host in `parsePullRequestURL`. Do not loosen
  the two-segment rule for GitHub or Bitbucket.
- Does an empty `vcs_provider` stay GitHub? Yes. Existing rows depend on it.

## Step 4. Choose events

For each factory behavior, name the provider event, the SuperPlane trigger,
and the webhook scope.

- GitHub delivers repository events from a repository hook. The App webhook
  keeps installation and member events only. Two deliveries run a trigger
  twice.
- Bitbucket Forge delivers pull request and build events to SuperPlane
  routes under `pkg/public/bitbucket_forge.go`.
- GitLab creates a project hook through `pkg/integrations/gitlab`. Group
  hooks are a paid tier and would deliver the same event again.

Say who can create the hook, and what setup shows when that role is missing.

## Step 5. Map operations

For each `vcs.Provider` method, name the provider API and the response
fields. Add a method only when both existing hosts can implement it. Keep
host-specific calls next to the provider, as `bitbucketProvider` does.

Check these mappings. They are the usual breaks:

- Merge method names. GitHub uses `squash`, `merge`, and `rebase` on the
  call. Another host may store the method on the project and use a separate
  squash flag.
- Mergeability. Map the host status onto the existing blocked-reason names.
  Add a name only when no current name fits.
- Checks. Name the object that becomes one console row. GitHub uses a check.
  Bitbucket uses a build. GitLab uses a pipeline, plus commit statuses for
  external CI.
- Comments. Say whether acknowledgement is a reaction or a comment, and
  include the task link in the first feedback template.
- Review bots. If the host has no bot catalog, use a typed name field.
- Clone token scope. Say whether the write token is limited to the workspace
  repository. If it is, context repositories need a separate read token.
- Velocity author. Name the stable user id. Exclude bots by account type.
  Keep the SuperPlane agent if the host reports it as a bot.

## Step 6. List the files

Walk the file map in the specification. For this host, mark each row as
reuse, extend, or skip. Name the integration client you will call. Do not
add a second HTTP client when `pkg/integrations/<name>/` already has one.

Include:

- Provider constant, onboarding validation, capabilities, clone URL.
- Linked account connect and remove.
- Feature flag, while the host is unfinished.
- Onboarding API and the first-run screens.
- Templates and `ResetOrganizationBuiltInAutomations` coverage. Follow
  [pkg/grpc/actions/factories/AGENTS.md](../../pkg/grpc/actions/factories/AGENTS.md).
- Webhook route and signature check.
- Velocity worker branch.
- URL parsers, branch links, and Verify hand-off.
- Env vars, `.env.example`, `docker-compose.dev.yml`, and
  `docs/contributing/connecting-to-3rdparty-services-from-development.md`.
- The process that refreshes tokens. Integration sync runs in
  `pkg/workers/integration_request_worker.go`.

New tables use `make db.migration.create NAME=<name>`. Do not create a
migration file by hand. Do not write GitLab or Bitbucket rows into
`vcs_provider_installations`. That table is the GitHub installation catalog.

## Step 7. Set the release gate

Ship the host button behind an organization feature flag. The flag comes off
only after the slices you named are in place:

- Connect, switch account, and remove the linked account.
- Select a repository and open a pull request from Implement.
- Checks, feedback, close, merge, and mergeability.
- Velocity for a merged pull request.
- Verify hand-off for a URL on this host.

GitLab stays disabled on the host screen until the flag is on. An empty
hosted configuration keeps the manual dialog.

## Confirm, then implement

Repeat the plan in one short spec. Use these headings:

- Provider
- Connect model
- Repository identity
- Events and webhooks
- Operations
- Velocity identity
- Capabilities in the first slice, and the later slices
- Files to extend
- Differences from GitHub and Bitbucket
- Feature flag and the removal gate
- Decisions the human still owns

Ask the human to confirm. Start code only after they confirm. Implement one
slice at a time. Turn a capability on in the same slice as its operation.
