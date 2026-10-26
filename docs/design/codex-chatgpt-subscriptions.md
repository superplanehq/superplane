# Codex with ChatGPT Subscriptions (Bring Your Own Subscription)

This document is the product requirements doc (PRD) for running factory Codex
agents on a ChatGPT subscription instead of an API key.

- **Status:** Draft
- **Owner:** Mijat Miletić (@mytmlt)
- **Origin:** [#8061 R&D: Use vendor subscriptions](https://github.com/superplanehq/superplane/issues/8061)
- **Prior work:** [#8573](https://github.com/superplanehq/superplane/pull/8573) (proof of concept, closed),
  [#8693](https://github.com/superplanehq/superplane/pull/8693) (shared token and lease design, merged into the #8573 branch)
- **Implementation stack (draft):**
  [#8739](https://github.com/superplanehq/superplane/pull/8739) →
  [#8741](https://github.com/superplanehq/superplane/pull/8741) →
  [#8742](https://github.com/superplanehq/superplane/pull/8742)
- **Feature flag:** `codex_subscriptions` (organization experimental feature, not released)

## Decision Summary

- Phase 1 supports one vendor: a ChatGPT subscription used by the Codex CLI.
  Claude and Cursor subscriptions are out of scope for this phase.
- API-key BYOK stays unchanged. The subscription is an extra model source.
- A workspace (factory) connects a ChatGPT subscription once with a device
  sign-in. Later Codex runs in that workspace reuse it.
- The OpenAI refresh token stays in SuperPlane. Each runner gets a random
  per-run lease secret instead. Codex refreshes through a SuperPlane endpoint.
- Sign-in and model-catalog refresh are exclusive. Runs share the connection
  and are not capped by this feature.
- Model usage on a subscription costs zero in SuperPlane. Hosted compute stays
  billable.
- In this phase, only organizations with exactly one active human member can
  connect a subscription. Multi-member organizations are an open question.
- The feature stays behind `codex_subscriptions` until an internal
  organization verifies a live sign-in.

## Problem

Organization BYOK and onboarding connect Anthropic, OpenAI, and OpenRouter with
an API key. Many users already pay for a Claude, Codex (ChatGPT), or Cursor
subscription. Those users do not have an API key, so they cannot bring their
own access to SuperPlane.

Issue #8061 asks SuperPlane to:

- keep API-key BYOK,
- investigate OAuth, CLI login, or other vendor subscription auth, and
- decide what SuperPlane can ship without asking for an API key.

The Codex CLI supports a ChatGPT device sign-in and stores an OAuth bundle in
`$CODEX_HOME/auth.json`. That makes Codex the first path that SuperPlane can
ship. The PoC (#8573) proved the path end to end. #8693 removed its main
limit: two runners that refresh the same refresh token sign the account out.

## Goals

- A user can run factory Codex agents on their ChatGPT subscription without an
  API key.
- The user signs in once per workspace. Runs reuse the sign-in without user
  action.
- Concurrent runs on one subscription do not sign the account out.
- The real refresh token never reaches a runner VM.
- The user can choose a model and reasoning level from the models that their
  subscription allows.
- The user can see the last known subscription quota.
- Spend reports show subscription usage separately from hosted and BYOK usage.
- The rollout is gated and reversible.

## Non-goals

- Claude, Cursor, or other vendor subscriptions (tracked in #8061).
- Sharing one subscription across members of a multi-member organization in
  this phase.
- Changes to API-key BYOK or SuperPlane-hosted models.
- Limiting how many runs can use one subscription at the same time.
- Enforcing or predicting ChatGPT quota before a run starts.
- Live quota updates. Phase 1 shows saved snapshots only.

## Users and Use Cases

| User | Use case |
| --- | --- |
| Solo builder with ChatGPT Plus/Pro | Sets up a workspace and picks "Use your ChatGPT subscription" instead of a key or hosted models. |
| Same user, later | Starts several tasks in parallel. All run on the same subscription. |
| Same user, quota check | Opens **Settings → Account → Codex usage** to see remaining quota and reset times. |
| Same user, token revoked | Sees "Connect ChatGPT again" and repeats the device sign-in. |
| Billing reviewer | Sees "ChatGPT subscription" as its own funding source with zero model cost. |

Supported ChatGPT plan types are TBD. The usage page shows the plan type that
Codex reports.

## Requirements

### Connection and sign-in

- **R1.** An eligible user can start a ChatGPT device sign-in from workspace
  setup (`POST /api/v1/factories/{factory_id}/codex-connection`).
- **R2.** SuperPlane runs `codex login --device-auth` on a temporary runner
  (`e1-large-amd64`, 15-minute timeout). The UI shows the verification URL and
  code that the runner reports.
- **R3.** The UI tells the user to allow device sign-in in ChatGPT Security
  settings.
- **R4.** The user can cancel a pending sign-in
  (`POST …/codex-connection:cancel`).
- **R5.** The user can disconnect (`DELETE …/codex-connection`). Disconnect
  fails while any task holds a lease ("stop the running tasks before you
  disconnect ChatGPT").
- **R6.** One connection exists per workspace. It records an owner user and an
  account label. Status is one of `pending`, `finishing`, `ready`,
  `reconnect_required`, or `disconnected`.
- **R7.** `GET …/codex-connection` returns status and never returns
  credentials.
- **R8.** Eligibility in this phase: the organization has the flag enabled and
  has exactly one active human member, and the caller is that member.
- **R9.** Once a workspace uses a subscription, its model source cannot switch
  ("Model source cannot change"). The way back to API-key or hosted models is
  TBD.

### Running on runners

- **R10.** The Codex runner component accepts "ChatGPT subscription" as a
  credentials source next to the existing key and integration options.
- **R11.** A subscription run adds a "Restore ChatGPT sign-in" step. The step
  writes an auth bundle to `$CODEX_HOME/auth.json` with mode `0600`.
- **R12.** A subscription run removes `OPENAI_API_KEY` and `OPENAI_BASE_URL`
  from the Codex environment.
- **R13.** The runner script redacts token and secret values from its output.
- **R14.** A run that starts while a sign-in or catalog refresh holds the
  connection stays pending and retries. It does not fail.
- **R15.** A run without a `ready` connection fails with "connect ChatGPT before
  this task can run".

### Token lifecycle and concurrency

- **R16.** SuperPlane stores the auth bundle encrypted
  (`factory_codex_connections.bundle_ciphertext`).
- **R17.** Each run gets a lease in `factory_codex_leases` with purpose `run`.
  Its random secret replaces the refresh token in the runner's `auth.json`.
  SuperPlane stores only the secret's hash and ciphertext.
- **R18.** Codex refreshes through
  `POST /api/v1/runner/codex-subscription/refresh`. The runner sets
  `CODEX_REFRESH_TOKEN_URL_OVERRIDE` to that URL.
- **R19.** SuperPlane calls OpenAI only when the access token expires within
  10 minutes. Otherwise it returns the stored access token. The refresh runs
  under a row lock, so concurrent runs cause one OpenAI call.
- **R20.** A permanent rejection (HTTP 401, `refresh_token_expired`,
  `refresh_token_reused`, or `refresh_token_invalidated`) sets the connection
  to `reconnect_required`. A transient failure returns
  `503 temporarily_unavailable` and keeps the connection.
- **R21.** Sign-in and catalog refresh take an exclusive `login` lease (one per
  connection, enforced by a unique index). A login lease blocks new runs. Run
  leases do not block each other.
- **R22.** A run lease ends when its task is terminal. A login lease ends only
  after Fleet Manager confirms that the runner VM is deleted
  (`runners.provider_deleted_at`, #8739).
- **R23.** A lost login runner sets the connection to `reconnect_required`. A
  lost run does not change the shared subscription.
- **R24.** Runner cancellation reaches busy runners (#8739).

### Model selection

- **R25.** SuperPlane reads the account's models from the Codex app-server
  `model/list` call at sign-in and on catalog refresh. It stores them in
  `factory_codex_connections.available_models`.
- **R26.** If a ready connection has no models, the UI requests a catalog
  refresh (`refresh_catalog: true`) without a new device sign-in.
- **R27.** Subscription models appear in the workspace model picker. The LLM
  Models settings page lists them as read-only ("All models available to this
  ChatGPT subscription are enabled").
- **R28.** The selected model goes to Codex with `-m`. The reasoning level goes
  with `-c model_reasoning_effort=`. Allowed levels: `minimal`, `low`,
  `medium`, `high`, `xhigh`, or empty for the default.
- **R29.** Without a selection, Codex uses its default model (#8741 behavior).
- **R30.** The run result records the model and effort that Codex actually used.
  The source is `turn_context.model` and `effort` in
  `$CODEX_HOME/sessions/**/rollout-*-<thread_id>.jsonl`.

### Usage and quota visibility

- **R31.** SuperPlane saves a quota snapshot (`usage_limits`,
  `usage_observed_at`) at sign-in, on catalog refresh, and at the end of each
  successful run ("Save ChatGPT usage" step). The PoC also took a snapshot at
  run start. Whether to restore that is TBD.
- **R32.** A missing or invalid quota snapshot never blocks a credential save.
- **R33.** `GET /api/v1/me/codex-usage` returns snapshots for the caller's own
  connections in the current organization.
- **R34.** **Settings → Account → Codex usage** shows each limit's primary and
  secondary windows: remaining percent, window length, and "Update needed"
  after a reset passes. It also explains that usage outside SuperPlane changes
  the remaining quota.

### Billing

- **R35.** Usage events have funding source `subscription`. The provider cost
  and billed amount are zero. The pricebook version gets a `+subscription`
  suffix.
- **R36.** Hosted compute for subscription runs stays billable.
- **R37.** Spend reports group "ChatGPT subscription" separately. It never
  counts as hosted spend or BYOK spend.
- **R38.** Whether the sign-in and catalog-refresh runner time counts as billable
  compute is TBD.

### Feature flag and rollout

- **R39.** All codex-connection API routes require the `factories` and
  `codex_subscriptions` features. The runner endpoints check eligibility again.
- **R40.** The flag label is "ChatGPT Subscription". The flag is off by default
  and is not marked released.
- **R41.** Turning the flag off makes the subscription unavailable to new runs
  (`ErrCodexConnectionUnavailable`). The effect on runs that are already in
  flight is TBD.

## UX Overview

1. **Workspace setup → agent screen.** "Use your own key or subscription"
   shows **Use your ChatGPT subscription** ("Sign in once. SuperPlane reuses
   that subscription on Codex runs.").
2. **Device sign-in.** Connect → "Wait for the ChatGPT sign-in code" → URL and
   code with "Open this link and enter the code" → Cancel sign-in.
3. **Finishing.** "ChatGPT sign-in is saved. Wait for connection setup to
   finish." After 60 seconds, a note says that setup is delayed and that the
   user can choose another model source.
4. **Ready.** "ChatGPT is connected as {account}." Reconnect shows "Connect
   ChatGPT again."
5. **Settings → LLM Models.** A "ChatGPT" badge and the read-only subscription
   model list.
6. **Agent model field.** Model plus Thinking level (`minimal`–`xhigh` for
   OpenAI and Codex).
7. **Settings → Account → Codex usage.** Quota cards. The Storybook states are
   `Connected`, `NoConnection`, `UsageUnavailable`, and `ResetPassed`
   (`Pages/Factories/Settings/Account/Codex usage`).

## Technical Design Summary

**Data**

- `factory_codex_connections`: one row per factory. It holds the owner user,
  status, verification URL and code, account label, encrypted bundle, and the
  `credential_version`, `final_checkpoint_version`, and
  `reservation_generation` counters. #8742 adds `usage_limits`,
  `usage_observed_at`, and `available_models`.
- `factory_codex_leases`: one row per task. Its purpose is `login` or `run`.
  Each row has a generation and a secret hash and ciphertext. A partial unique
  index allows one `login` lease per connection.
- `runners.provider_deleted_at`: set by Fleet Manager through
  `POST /admin/api/installation/fleets/{fleet_id}/runners/{runner_id}:provider-deleted`
  (#8739).

**Runner endpoints** (scoped JWT in `SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN`; the
refresh endpoint authenticates by lease secret)

- `GET /api/v1/runner/codex-subscription/credentials`: returns the auth bundle
  with the lease secret in place of the refresh token.
- `POST /api/v1/runner/codex-subscription/progress`: device URL and code.
- `POST /api/v1/runner/codex-subscription/checkpoint`: login and refresh
  leases save the real bundle. Run leases save only usage and models.
- `POST /api/v1/runner/codex-subscription/refresh`: the OAuth refresh proxy to
  `https://auth.openai.com/oauth/token`.

**Runner script:** `codex_subscription.js` provides
`login | restore | refresh | checkpoint | usage`.

**Cleanup:** `RunnerCleanupWorker` releases leases whose tasks are terminal and
whose VMs are confirmed deleted. It marks lost login runners as uncertain.

**Runner image:** Codex CLI 0.144.5 → 0.163.0. The image also passes the
SuperPlane MCP env vars and approval mode (#8739).

## Rollout Plan

1. **#8739: confirm runner deletion and deliver cancellation.** Deploy the
   server before Fleet Manager. Verify an API-key Codex run and a cancellation.
2. **#8741: reuse ChatGPT subscriptions on Codex runners.** It depends on #8739
   and keeps the Codex default model. Enable `codex_subscriptions` for one
   internal single-member organization, then test a live device sign-in.
3. **#8742: model selection and usage.** It depends on #8741. Review the
   Storybook usage states. Verify model selection, reasoning levels, and quota
   snapshots on the internal organization.
4. **Wider enablement:** the criteria and the decision to release the flag are
   TBD. The open questions below gate them.

Proposed verification checklist for the internal organization:

- [ ] Device sign-in reaches `ready` and shows the account label.
- [ ] Two or more concurrent runs succeed past an access-token refresh without
      a sign-out.
- [ ] A runner VM's `auth.json` holds no OpenAI refresh token.
- [ ] Revoking access in ChatGPT leads to `reconnect_required` and a reconnect
      prompt.
- [ ] Spend shows "ChatGPT subscription" with zero model cost and billed
      compute.
- [ ] Cancel and disconnect behave as R4 and R5 describe.

## Success Metrics (proposed)

These metrics are proposals. Neither the targets nor the instrumentation are
agreed yet.

- Sign-in completion rate (started → `ready`). Target TBD.
- Share of subscription runs that fail on credentials (`reconnect_required`,
  refresh 401 or 503, not ready). Target: near zero outside real revocations.
- Unplanned `reconnect_required` transitions per connection per week.
- Number of single-member orgs that pick the subscription over hosted or BYOK
  at onboarding.
- Hosted compute minutes from subscription runs (revenue proxy).

## Risks

- **Terms of service and compliance.** It is unclear whether vendor terms allow
  automated or server-side use of a personal ChatGPT subscription, or sharing
  one across an organization. This is an open question, not a settled answer.
- **Undocumented interfaces.** The flow depends on Codex CLI behavior:
  `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, the `auth.json` layout, app-server
  `model/list` and rate limits, rollout files, the OAuth client ID, and the
  token endpoint. A Codex upgrade can break it. The image is pinned to
  `0.163.0`.
- **Credential custody.** SuperPlane now holds user OAuth refresh tokens. A
  login runner VM briefly holds the real bundle.
- **Quota exhaustion.** Unbounded concurrent runs can drain the subscription
  quickly. The visible behavior when the quota runs out is TBD.
- **Stale usage.** Snapshots miss usage outside SuperPlane. The UI says so.
- **Login blocks runs.** A stuck sign-in holds the exclusive lease until the VM
  deletion is confirmed. Queued runs wait until then.
- **Review feedback.** Greptile flagged a refresh-route lease race on #8693.
  Whether the split stack fully resolves it is TBD.

## Open Questions

1. **Multi-user organizations.** Can several members share one subscription?
   How does a multi-member org set this up (one owner, or one subscription per
   member)? This is the named next step and is unresolved.
2. **Terms of service.** Do OpenAI terms allow this use, and the
   organization-wide sharing in (1)? This needs a legal or vendor answer.
3. **Supported plans.** Which ChatGPT plans support Codex device sign-in and
   give usable quota? TBD.
4. **Leaving a subscription.** Model-source switching is blocked today. What is
   the supported way back to API-key or hosted models?
5. **Quota exhaustion.** How should runs fail, and what should the user see?
6. **Start-of-run snapshot.** Should SuperPlane restore the PoC's start-of-run
   quota snapshot, or are end-of-run snapshots enough?
7. **Sign-in compute.** Is runner time for sign-in and catalog refresh
   billable?
8. **Flag-off behavior.** What should happen to in-flight runs when the flag is
   turned off?
9. **Other vendors.** When should Claude and Cursor subscriptions be evaluated
   (#8061)?

## Decisions Log

| Date | Decision | Source |
| --- | --- | --- |
| 2026-10-02 | Research vendor subscriptions and keep API-key BYOK. | #8061 |
| 2026-10-15 | PoC: reuse a ChatGPT subscription on Codex runners behind `codex_subscriptions`. Single-member orgs only. Model cost zero, compute hosted. | #8573 |
| 2026-10-20 | Keep the refresh token in SuperPlane, give each run a lease secret, and proxy refresh. Sign-in and model refresh stay exclusive. Runs are not capped. | #8693 |
| 2026-10-21 | Split the PoC into three PRs: runner lifecycle, then core subscription, then models and usage. Deploy the server before Fleet Manager. | #8739, #8741, #8742 |
| 2026-10-22 | Close the PoC PR in favor of the split stack. | #8573 |
| 2026-10-24 | Claude and Cursor subscriptions are out of scope for phase 1. | Owner |
