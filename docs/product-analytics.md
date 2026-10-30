# Factory product analytics

Factory product analytics measures SaaS acquisition, activation, activity, and
retention. The server sends events only when `POSTHOG_KEY` is set. It sends the
events to `https://us.i.posthog.com`.

The browser sends the signup event and identifies the account. The server sends
all Factory lifecycle events after the related operation succeeds. PostHog
autocapture, page views, page leaves, surveys, and session recording are
disabled.

## Data rules

All server events use schema version `1`. They contain `actor_type` and `source`.
Organization-scoped events contain `organization_id`. Factory and task events also contain `factory_id`
and `task_id` when these identifiers are available. Each event has the PostHog
group `organization` set to the organization ID when an organization is present.
Only `account_deleted` can omit the organization ID and group. Accounts with no
organizations still produce an account event. Other events require organization context.

The `actor_type` value is `user`, `system`, or `external`. An authenticated
user uses the global account UUID as the PostHog distinct ID. System and
unmapped external actors use hashed, organization-scoped synthetic IDs. A
mapped GitHub user uses the matching account UUID.

For `organization_created`, the server includes the creator email and name
from the authenticated account. It omits empty or whitespace-only identity
fields.
The event contains these values as top-level properties and in `$set`.
PostHog uses `$set` to update the person properties through the same capture.
This applies to initial onboarding and later organization creation.
It does not depend on a browser identity call.

Browser identification continues to supply account identity for browser events.
It uses the same global account UUID as the server.

Do not add task content, prompts, repository names, URLs, branches, pull
request titles, errors, tokens, or credentials to an event.
Do not add provider method IDs, card metadata, or payment details to an event.

## Event catalog

| Event | Successful outcome | Event properties |
| --- | --- | --- |
| `user_signed_up` | A new account returns through the signup callback. | Initial UTM properties and `product_updates_opt_in` when available. |
| `organization_created` | The first or a later organization commits. | `creation_path`: `initial_onboarding` or `workspace_switcher`; creator `email` and `name` when nonblank; `$set` with the same identity fields. |
| `account_deleted` | Self-service or installation-admin account deletion commits. | `account_id`, `actor_account_id`, `deletion_source`, `deleted_at`, and `deleted_organization_ids`. |
| `organization_deleted` | Direct or eligible account-driven organization deletion commits. | `actor_account_id`, `deletion_source`, `deleted_at`; `account_id` for account-driven deletion. |
| `factory_deleted` | Explicit factory deletion commits and hides its tasks. | Organization and factory identifiers. |
| `payment_method_removed` | A complete billing read observes usable saved-card availability change to unusable. | `provider`, `detection_method`, `reason`, `observed_at`, and `transition_id`. |
| `factory_onboarding_completed` | Factory onboarding first changes from incomplete to complete. | `vcs_provider`, `issues_source`, and `agent_harness` enum values. |
| `factory_task_created` | A task commits. | `creation_source`: `manual`, `issue_import`, `automation`, `handoff`, or `mcp`; `issue_source` enum for imports. |
| `factory_task_refined` | The first human planning interaction in a planning session succeeds. | `interaction`: `message` or `survey_answer`. |
| `factory_task_started` | A task dispatch commits. | `run_kind`: `initial` or `rerun`. |
| `factory_task_approved` | A task first closes as completed through approval. | `close_source`. |
| `factory_task_rejected` | A task first closes with a rejected result. | `rejection_source`. |
| `factory_task_archived` | A task first closes as archived. | `archive_source`. |
| `factory_task_merged` | A pull request first changes to merged. | `provider` and `merge_source`. |

The server event wrapper accepts only the server event names in this
table. The browser emits only `user_signed_up`.

`run_kind` is `initial` when the task has no earlier line dispatch. A later
dispatch uses `rerun`, including after the task returns to draft. Automated
imports set `issue_source` to the intake source enum.

## Deletion and archive events

Account deletion uses the acting account UUID as the distinct ID. `account_id`
identifies the deleted account. `actor_account_id` identifies the authenticated actor.
`deletion_source` is `self_service` or `admin`. The server preserves account
identity before deletion removes memberships or changes account fields.
`deleted_at` is the committed deletion timestamp in UTC.
`deleted_organization_ids` lists only organizations that the operation deleted.

Organization deletion uses `deletion_source: direct` or `account_deletion`.
Account-driven deletion includes the deleted `account_id`. Existing eligibility
rules still apply. An organization with another owner produces no deletion event.
Organization events include the organization UUID and PostHog organization group.
Deletion event IDs derive from the event name and deleted entity UUID.

Explicit factory deletion produces one parent event. Organization and account
deletion events explain dependent task disappearance without per-task events.
The current factory API has no separate task-delete action.

Task archive events use `archive_source: manual`, `automation`, or `backlog_refresh`.
Backlog refresh emits only after each archive commits. Failed source reads,
unchanged tasks, and failed archive transactions produce no archive event.
Archive IDs retain the existing first-archive convention: event name, task UUID, and the archived result.
Repeated refreshes do not produce additional archive events for the same archive.

## Observed saved-card availability

`payment_method_removed` includes removal, expiration, and uncertain availability loss.
It records a change from a previously observed usable card to no usable cards.
The first complete observation establishes a baseline and produces no event.
Removing or expiring one card produces no event while another usable card remains.
Repeated unusable observations produce no additional event. Recovery permits a new loss event.

The server reuses existing Polar reads in billing description, billing sync,
and applicable plan selection. Detection occurs at a subsequent billing read.
`observed_at` is the observation time, not the exact provider removal or expiration time.
Opening the billing portal and unrelated subscription notifications do not record observations.
Execution payment guards remain read-only and enforce the existing card requirement.

The event uses `provider: polar`, `detection_method: availability_check`,
`actor_type: system`, and `source: polar_api`. Its distinct ID uses the existing
organization-scoped synthetic identity convention with the identity `billing`.
The request caller is an observer, not a proven removal actor.
The event includes the organization UUID and its PostHog organization group.

The `reason` value is:

- `removed`: all previously usable method identities are absent from the complete list.
  This means observed disappearance. It does not prove a user deleted a card.
- `expired`: all previously usable methods remain present with valid, expired card expiration values.
  A normal card stays usable through its expiration month in UTC.
- `unknown`: mixed disappearance and expiration, invalid metadata, or another ambiguous loss.

Valid Korean cards retain the existing usable-card rule.
Missing customers, provider failures, and incomplete lists do not update the baseline.
The server reads all pages before it accepts an observation.
It serializes updates per organization and rejects reads that started before the latest accepted read.
The database stores availability, read ordering, observation time, organization-scoped method hashes,
and the last transition UUID. It stores no raw method IDs or card metadata.
Organization hard deletion removes this state through a foreign-key cascade.

Each committed loss receives a new `transition_id`. The event ID derives from
the event name, organization UUID, and transition UUID. Recovery followed by loss
produces a different ID. A later plan-selection rejection does not undo the observation.
Observation storage and analytics failures do not block the billing operation.
Analytics capture occurs only after the observation transaction commits.

These events provide prospective tracking. They do not reconstruct historical
deletions, count every removed card, or identify who removed one.
The API reuses existing Polar and PostHog configuration. No webhook subscription,
scheduled polling, or new worker dependency is required. Apply the database migration
before deploying the API change.

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
and 30-day windows. Count an account when that person refines, starts,
approves, rejects, archives, or merges a task. Signup, organization creation,
onboarding, and task creation do not count. Use these events:

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
