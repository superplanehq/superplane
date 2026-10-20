# Factory product analytics

Factory product analytics measures SaaS acquisition, activation, activity, and
retention. The server sends events only when `POSTHOG_KEY` is set. It sends the
events to `https://us.i.posthog.com`.

The browser sends the signup event and account identity. The server sends all
Factory lifecycle events after the related operation succeeds. PostHog
autocapture, page views, page leaves, surveys, and session recording are
disabled.

## Data rules

All server events use schema version `1`. They contain `organization_id`,
`actor_type`, and `source`. Factory and task events also contain `factory_id`
and `task_id` when these identifiers are available. Each event has the PostHog
group `organization` set to the organization ID.

The `actor_type` value is `user`, `system`, or `external`. An authenticated
user uses the global account UUID as the PostHog distinct ID. System and
unmapped external actors use hashed, organization-scoped synthetic IDs. A
mapped GitHub user uses the matching account UUID.

Do not add task content, prompts, repository names, URLs, branches, pull
request titles, errors, tokens, or credentials to an event.

## Event catalog

| Event | Successful outcome | Event properties |
| --- | --- | --- |
| `user_signed_up` | A new account returns through the signup callback. | Initial UTM properties and `product_updates_opt_in` when available. |
| `organization_created` | The first or a later organization commits. | `creation_path`: `initial_onboarding` or `workspace_switcher`. |
| `factory_onboarding_completed` | Factory onboarding first changes from incomplete to complete. | `vcs_provider`, `issues_source`, and `agent_harness` enum values. |
| `factory_task_created` | A task commits. | `creation_source`: `manual`, `issue_import`, `automation`, `handoff`, or `mcp`; `issue_source` enum for imports. |
| `factory_task_refined` | The first human planning interaction in a planning session succeeds. | `interaction`: `message` or `survey_answer`. |
| `factory_task_started` | A task dispatch commits. | `run_kind`: `initial` or `rerun`. |
| `factory_task_approved` | A task first closes as completed through approval. | `close_source`. |
| `factory_task_rejected` | A task first closes with a rejected result. | `rejection_source`. |
| `factory_task_archived` | A task first closes as archived. | `archive_source`. |
| `factory_task_merged` | A pull request first changes to merged. | `provider` and `merge_source`. |

The server event wrapper accepts only the nine server event names in this
table. The browser emits only `user_signed_up`.

## Metrics

The user conversion funnel is:

`user_signed_up` → `organization_created` →
`factory_onboarding_completed` → imported `factory_task_created` →
`factory_task_started` → `factory_task_merged`.

The organization conversion funnel starts with `organization_created` and
uses the same later steps. Group it by `organization_id`.

User activation is the identified account's first `factory_task_merged`.
Organization activation is the organization's first `factory_task_merged`.

Week-one retention requires another `factory_task_merged` from 7 through 13
days after activation. Calculate user and organization retention separately.

WAU and MAU are unique account IDs with `actor_type = user` over rolling 7-day
and 30-day windows. Use these task events:

- `factory_task_created`
- `factory_task_refined`
- `factory_task_started`
- `factory_task_approved`
- `factory_task_rejected`
- `factory_task_archived`
- `factory_task_merged`

WAO and MAO are unique `organization_id` values over the same windows. Count
task refinement and lifecycle activity. Exclude an organization when its only
event is automated intake.

Website and documentation visits are not part of this catalog. Therefore, the
acquisition funnel starts at signup until those surfaces implement compatible
tracking.

## Release cleanup

After staging verification, archive the legacy PostHog dashboards and actions.
Tag their event definitions as `legacy` and record the release date. Do not
delete historical events. Confirm that no legacy, survey, autocapture,
page-view, page-leave, or replay events arrive after the release.
