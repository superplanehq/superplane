# Retire the legacy canvas assistant

Repository removal does not complete live cleanup. An administrator must confirm
Claude resource retirement and APT publisher revocation within this task.
Keep resource identifiers and dispositions in the task. Never record credential
values, vault contents, customer prompts, or downloaded session payloads.

## Rollout order

1. Block legacy `/api/v1/agents/*` and `/ws/agents/sessions/*` traffic at ingress.
2. Stop legacy stream workers. Set `START_AGENT_STREAM_WORKER=no` on old processes
   during the transition. Stop their RabbitMQ consumers and in-flight streams.
   Remove dedicated `agent-stream-requested` consumers and bindings after shutdown.
   Remove `agent-session-event` subscriptions from the shared canvas exchange.
   Confirm queue ownership before deleting a queue. Preserve shared exchanges and queues.
3. Capture identifiers before running the destructive migration.
4. Deploy removal to every API and worker replica. Confirm all old replicas stop.
5. Retire hosted resources with administrator credentials while those credentials remain available.
6. Apply `remove-legacy-built-in-agent` after the inventory is complete and all old processes stop.
7. Remove assistant-only deployment credentials. Confirm no replica can recreate hosted resources.

The affected processes are API/server startup, the legacy stream worker, canvas
cleanup, organization cleanup, and factory runtime cleanup. Runners and planning
sessions continue through their supported implementations.

## Identifier inventory

Read `ANTHROPIC_AGENT_ID` and `ANTHROPIC_ENVIRONMENT_ID` from each installation's
configuration. Record the secret identifier for `ANTHROPIC_API_KEY`, not its value.
Before the migration, export only these local identifiers:

```sql
SELECT id, organization_id, user_id, canvas_id, provider, provider_session_id
FROM agent_sessions
ORDER BY provider, provider_session_id;
```

Local rows are incomplete. Inspect the provider account for replaced, archived,
and older sessions associated with this agent and environment. Inventory every
page of sessions, files, environments, agents, and vault references.

The removed provider used `DELETE /sessions/{id}` for deletion and
`POST /sessions/{id}/archive` for archiving. These operations are different.
Delete owned sessions through the supported provider operation. Archive the agent
if that is the provider-supported retirement operation. Record any retention limits.

Reference uploads used filenames with an uppercase normalized source name and a
12-character SHA-256 suffix. Sources included `SUPERPLANE_APP_BUILDER_*`,
`SUPERPLANE_CLI_*`, `SUPERPLANE_MONITOR_*`, `CONSOLE_AND_WIDGETS`,
`RICH_UI_WIDGETS`, and generated component references. Inspect old versions too.
Names alone do not prove ownership. Confirm mounts and consumers before deletion.

Retire the assistant's environment, uploaded files, vault credentials, scoped CLI
tokens, and legacy signing-key copies. Preserve customer-configured Claude agents,
shared environments and keys, workspace resources, and supported runner credentials.
Confirm expiry or revocation of outstanding scoped CLI tokens. Preserve shared JWT signing keys.

## Deployment sources

- Local Compose, Coolify source Compose, and `.env.example` remove the
  assistant's process configuration.
- The self-hosted chart under `release/superplane-helm-chart/helm` contains no
  assistant variable references.
- Launchpad has no `ANTHROPIC_AGENT_ID`, `ANTHROPIC_ENVIRONMENT_ID`, or
  `ANTHROPIC_API_KEY` references in the inspected tracked deployment sources.
- `superplanehq/helm-chart` is inaccessible to this run. A maintainer must inspect
  API and worker environment values and Secret references there before rollout.

Remove shared `ANTHROPIC_API_KEY` credentials only after confirming every consumer.
Customer integration and hosted runner credentials use separate supported paths.

## Public APT retirement

The [companion Launchpad branch](https://github.com/superplanehq/launchpad/tree/chore/retire-public-apt-publishing)
removes `.semaphore/publish-apt.yml` (commit `1a9e1744`). The branch is pushed.
It is not merged, and live publishing configuration is not verified.
No surviving tracked release entry point references it. Its main Semaphore
pipeline has no APT promotion. npm and direct-download publishing remain active.
Update external installation documentation at `docs.superplane.com/installation/cli`
if it still recommends APT.

Before merging the companion change, disable live manual, release, scheduled,
and external APT triggers. Stop in-flight APT jobs. Confirm no active publisher
can update public package storage. Existing package storage can remain.

Inspect the Semaphore secret `aws-apt-repo` and these identifiers:

- AWS publisher principal and access-key identifiers
- `APT_S3_BUCKET` publisher permissions
- `GPG_KEY_FILE` signing-key fingerprint and all private-key copies
- `GPG_PASSPHRASE` secret entry
- Legacy copies in Claude vaults and environments

Revoke dedicated publisher permissions and signing material after confirming
consumers. Delete the whole secret only if every entry is dedicated to APT.
Preserve shared AWS access and unrelated release credentials.

## Administrator confirmation

For each resource or credential, record:

| System | Identifier | Ownership evidence | Disposition | Confirmed by/date | Retention limit |
| --- | --- | --- | --- | --- | --- |

Use `deleted`, `archived`, `revoked`, or `retained/shared` as the disposition.
Name unresolved resources explicitly. Separate repository merge status from live
deployment and provider status. This run has not performed live cleanup.
The task remains incomplete until the administrator supplies both confirmations.

## Repository verification evidence

The test database migrated with populated legacy sessions and messages. A factory
planning session and a historical Anthropic usage row survived unchanged.
The legacy feature entry was removed, and the surviving feature stayed enabled.
The verification removed only its own fixture records afterward.

The migration regression test also checks the message foreign-key dependency,
canvas preservation, and permission cleanup in an isolated transaction.
Route tests exercise authenticated requests and require no registered legacy handler.

Backend and UI builds, formatting, lint, and protobuf field-number checks pass.
Focused cleanup, model, public-route, feature, and authorization tests pass.
The factory actions suite passes all 1,494 tests. Runner and Claude integration
suites pass; 18 media and Docker tests skip because the app test container lacks
ffmpeg or Docker. Launchpad validation passes all 22 remaining pipeline files.

A broader backend test command exceeded 60 seconds and was stopped. Smaller
suites replaced it. Full backend and UI suites were not run under the automation's
test-duration constraint. Live provider cleanup, publisher revocation, external
installation docs, and the separate hosted chart still need administrator confirmation.
