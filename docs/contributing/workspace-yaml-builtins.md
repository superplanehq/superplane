# Workspace YAML

The v2 Factory manifest at `.superplane/main.yml` configures an existing
workspace. Export includes repositories, stages, and automations. Keep the
document-level `kind: Factory`. Stage entries do not include `kind` fields.

Select the destination with `--workspace` for export, plan, and apply:

```sh
superplane workspaces export --workspace engineering
superplane workspaces plan --workspace engineering
superplane workspaces apply --workspace engineering
```

Apply requires an existing workspace. Workspace creation and multiple workspaces
in one `.superplane` folder are not supported.

The top-level `name` must match the selected workspace. When copying YAML to a
workspace with a different name, update `name` before you run plan.

For example, an exported stage entry can contain:

```yaml
key: verify
name: Verify
automations: []
```

SuperPlane matches each stage to the live workspace by its stable `key` and
uses the live stage kind. A stage name does not determine its kind. Keep
the exported keys when you rename stages.

Stage `kind` is not a supported YAML field. SuperPlane uses the live kind for
stage order, phase settings, and runner validation.

Include every existing stage exactly once. YAML cannot add or remove stages.
Backlog must remain first, and Done must remain last. Keep existing `run`
bindings. Only stages with a runner binding support `max_parallelism`.
Existing permissions for stage names and order still apply.

Export followed by plan produces no changes when the workspace has not changed.

## Repositories

Use the optional workspace-level `repositories` section to configure the primary
repository, additional repositories, and their existing access settings:

```yaml
repositories:
  binding:
    integration_name: "Engineering GitHub"
  primary:
    repository: acme/application
    ref: main
  backlog_repository: acme/backlog
  additional:
    - repository: acme/shared-library
      ref: main
      access: write
      enabled: true
    - repository: acme/reference-docs
      access: read
      enabled: false
```

When `repositories` is present, include its connection and primary repository.
The primary repository remains writable. Each additional entry requires
`repository` and explicit `access: read` or `access: write`. `enabled` defaults
to `true`. An omitted `ref` uses the provider default branch.

An additional entry can set `provider`; it defaults to the workspace connection
provider. Existing provider and authentication restrictions still apply.
Repository access retains the current runtime behavior. YAML does not add
credential isolation or support a read-only primary repository.

`backlog_repository` defaults to the primary repository. Set it only when Backlog
uses a different repository. GitHub Issues intake without an explicit
`binding.resource_id` uses the desired Backlog repository.

`additional` is the complete desired list. An omitted or empty list removes all
additional repositories. Other agent resources remain unchanged. SuperPlane
matches existing entries by provider and canonical repository name to retain
record identity. Export includes disabled entries and explicit access values.

If `repositories` is absent, apply preserves the existing repository settings.
A workspace without a primary repository keeps its existing export behavior.
Repository settings apply to the whole workspace, including listeners on other
workspace lines. Existing tasks keep their saved repository and branch when
workspace defaults change.

### API credentials and GitHub repository access

Personal API tokens act as their owners. Repository selection requires the
owner's existing workspace permissions and the same GitHub access as the UI.
A personal token can select a new accessible repository.

Organization API keys use their existing permissions and scope. They do not
require a personal GitHub account or use their creator's GitHub access.
For hosted GitHub connections, a key can select only repositories already
granted to the selected connection in the same organization.
A repository in the same installation still requires its own grant.
Removed repositories and suspended installations are unavailable.

These rules cover primary, Backlog, and additional repositories, including
branch verification. Repository APIs without a connection selection resolve an
existing granted connection in the organization. Organization keys cannot create
connections or extend access to an entire installation.

If an organization key selects an ungranted repository, plan reports:

> This repository is not authorized for the selected connection. Use a personal API token with GitHub access to select it first.

Plan does not create grants. Apply checks current access again before it saves
changes. Repository grants follow existing workspace and automation uses.
Switching or removing those uses can remove grants; grants are not a permanent
repository allowlist. Bitbucket and customer-managed GitHub credentials retain
their existing authorization rules.

## Built-in installations

Use a built-in alias in `uses` to configure an installed automation or install
a missing automation. Deleted installations are missing. Apply creates new
canvases and supporting records; it does not restore deleted records.

The built-in catalog contains these aliases:

| Automation | Alias |
| --- | --- |
| Task analysis | `@superplane/task-analysis` |
| Scheduled intake | `@superplane/scheduled-intake` |
| Implement | `@superplane/implement` |
| Issue label intake | `@superplane/issue-intake` |
| Pull request closure | `@superplane/pr-closure` |
| Merge confidence | `@superplane/merge-confidence` |
| Review request | `@superplane/review-request` |
| Visual Evidence | `@superplane/visual-evidence` |
| Pull request discussion | `@superplane/pr-discussion` |
| Pull request checks | `@superplane/pr-checks` |
| GitHub issues | `@superplane/intake/github-issues` |
| Dependabot alerts | `@superplane/intake/dependabot-alerts` |
| Jira issues | `@superplane/intake/jira-issues` |
| Sentry exceptions | `@superplane/intake/sentry-exceptions` |
| PagerDuty incidents | `@superplane/intake/pagerduty-incidents` |
| Productive tasks | `@superplane/intake/productive-tasks` |
| Datadog errors | `@superplane/intake/datadog` |
| Linear issues | `@superplane/intake/linear-issues` |

Keep each automation `id` stable when its name changes. YAML installations
retain their workspace, line, stage key, and manifest ID across exports.
Existing installations without YAML ownership retain their exported identities.

## Bindings and defaults

An optional `binding` mapping selects existing connections and resources:

```yaml
binding:
  integration_name: "Production Sentry"
  resource_id: "backend"
```

Use either `integration_name` or `integration_id`, never both. Names match
exactly within the destination organization. Export uses a connection name when
it is unique; otherwise it uses the connection ID. Missing, ambiguous,
unavailable, or incompatible connections produce diagnostics at the binding field.

Bindings do not create connections or contain credentials. The selected
integration must belong to the organization, be ready, and support the built-in.
Resources must belong to that integration. Linear bindings can set `team_id`.
Unsupported binding fields fail plan at the corresponding YAML field.

Repository automations use the workspace repository and connection by default.
Export always includes the selected intake connection, including a connection
that came from workspace defaults. Other explicit connection, resource, and team
bindings remain in exported YAML and survive admin reset and repository refresh.
Older intake manifests can still inherit workspace connections.

To copy YAML to another configured workspace, create matching connection names
in its organization. ID bindings must reference destination connections.

Provider intake cannot infer provider-specific selections from the repository.
Select the provider integration and its required resource or team explicitly
when workspace defaults cannot resolve them. Provider restrictions and
experimental feature gates still apply.

| Built-in | Required selections |
| --- | --- |
| GitHub issues, Dependabot alerts, Issue label intake | Defaults to the workspace GitHub connection and Backlog repository |
| Jira issues | `integration_name` or `integration_id`, and `resource_id` with the project key |
| Sentry exceptions | `integration_name` or `integration_id`, and `resource_id` with comma-separated project slugs |
| Productive tasks | `integration_name` or `integration_id`, and `resource_id` with the project ID |
| Datadog errors | `integration_name` or `integration_id`, and `resource_id` with the service name |
| PagerDuty incidents | `integration_name` or `integration_id`, and `resource_id` with the service ID |
| Linear issues | `integration_name` or `integration_id`, and `team_id`; legacy project selection can use comma-separated IDs in `resource_id` |

Task analysis uses the workspace agent and connection; it does not accept a
repository resource override. Other repository templates use the application
repository by default. Bitbucket workspaces do not support GitHub issue intake,
Dependabot intake, Issue label intake, Review request, or Visual Evidence.
Scheduled intake and Review request require their existing experimental features.

Place intakes, Task analysis, Scheduled intake, and Issue label intake in
Backlog. Keep Implement in a phase and Pull request closure in Done.
Place the remaining built-ins in Verify.

Omitted settings use the workspace and fresh template defaults for both new
and existing installations. Explicit `false`, `0`, `""`, and `[]` remain explicit
values where the setting supports them. Export includes supported overrides
relative to a fresh installation. Older exports cannot recover settings they
never recorded.

Agent options remain under `with.agents`, keyed by the agent node ID from an
export. Supported options are `additional_instructions`, `model`,
`thinking_level`, `execution_timeout_seconds`, `skip_setup_commands`, and
`skip_repository_checkout`. Supported agents also accept `include_visual_evidence`.
Existing `with.additional_instructions` for Implement
remains compatible.

## Supported settings

Settings belong to the built-in that declares them. Plan rejects unknown fields,
wrong types, and settings from another provider at the offending YAML field.
Use booleans for toggles, integers for counts and percentages, and string lists
for filter selections. Names and IDs are strings.

All provider intakes accept `with.intake.confidence_pct` from 0 through 100 and
`with.intake.default_owner_id`. The owner must belong to the workspace organization.
GitHub, Sentry, Jira, Productive, Dependabot, and Linear intakes also accept
`with.intake.paused`, which defaults to `false`.
Each intake also accepts these provider settings:

| Intake | Additional `with.intake` fields |
| --- | --- |
| GitHub issues | `labels`, `label_filter_mode`, `assignment`, `new_issues`, `reopened_issues`, `authors_with_access`, `superplane_label_added` |
| Jira issues | `labels`, `label_filter_mode`, `assignment`, `new_issues`, `reopened_issues`, `jira_move_on_complete`, `jira_completion_column` |
| Sentry exceptions | `sentry_new_issues`, `sentry_regressed_issues`, `sentry_assigned_issues`, `sentry_levels`, `sentry_project_ids` |
| Productive tasks | `exclude_key_tasks`, `task_list_ids` |
| Dependabot alerts | `dependabot_severities` |
| Datadog errors | `datadog_triggered_alerts`, `datadog_retriggered_alerts`, `datadog_environments` |
| Linear issues | `linear_project_ids`, `linear_include_issues_without_project`, `linear_labels` |
| PagerDuty incidents | No additional settings |

`label_filter_mode` accepts `include` or `exclude`. `assignment` accepts `any`,
`assigned`, or `unassigned`. Sentry levels are `fatal`, `error`, `warning`,
`info`, and `debug`. Dependabot severities are `critical`, `high`, `medium`,
and `low`. Linear selects its team through `binding.team_id`.

Pull request feedback uses `with.pr_feedback`:

| Built-in | Supported fields |
| --- | --- |
| Pull request discussion | `mention`, `ignore_bots`, `address_all_bots`, `allowed_bots` |
| Pull request checks | `check_names`, `maximum_attempts`, `runner_integration_ids` |

Pull request checks require at least one check name. `maximum_attempts` accepts
integers from 1 through 10. Runner integration IDs reference existing ready
connections from the same organization.

Template settings use `with.template`:

| Built-in | Supported fields |
| --- | --- |
| Scheduled intake | `type`, `hour`, `minute`, `timezone`, `cron_expression`, `prompt`, `default_owner_id` |
| Implement | `draft_pull_requests` |
| Merge confidence | `enabled_checks`, `risk_rules`, `publish_pr_check`, `publish_results`, `results_body`, `results_comment` |
| Review request | `merge_confidence`, `fallback_reviewer`, `publish_pr_check`, `publish_review_findings`, `review_findings_body`, `review_findings_comment` |

Scheduled intake accepts `type: days` or `type: cron`. Its default daily
schedule runs at 09:00 with UTC offset `"0"`. `hour` accepts 0 through 23,
and `minute` accepts 0 through 59. `timezone` is a string UTC offset from
`"-12"` through `"14"`. Cron schedules also require `cron_expression`.
Omit `prompt` to use the current template proposal prompt.

`enabled_checks` selects supported Merge confidence check IDs; `[]` disables
all checks. `risk_rules` is a string. Result destinations use boolean settings.
Review request's `fallback_reviewer` is a string, and `merge_confidence` is the
portable reference described below.

## Plan, apply, and provisioning

Plan prepares repository settings before automations. A combined repository and
intake change uses the requested connection and repositories throughout the plan.
Plan validates the complete desired configuration without writes, webhooks,
historical imports, or external provisioning. Dependencies can refer to retained
or newly installed automations, regardless of manifest order.

Review request refers to Merge confidence with manifest identities:

```yaml
with:
  template:
    merge_confidence:
      stage: verify
      automation: confidence
```

The `stage` value is a stage key, and `automation` is its automation ID.
SuperPlane infers this dependency only when exactly one valid source exists.

Apply checks the plan token again before publication. Changes to relevant
workspace defaults, repository settings, resolved connections, resources, or
dependencies require a new plan. Apply saves repository records, connection
grants, graphs, placement, ownership, subscriptions, and provisioning requests
in one transaction. A synchronous setup error rolls back the full apply.

Repository changes refresh affected listeners across the workspace. Explicit
automation resource overrides keep their selected repository. Existing tasks
retain a snapshot of their repository and branch before defaults change.

A successful apply means configuration and provisioning requests were committed.
External provisioning can finish later. Datadog monitor changes use durable node
requests and existing worker retries. Notifications follow the transaction commit.
YAML-created intakes skip historical imports and record their import status as
`skipped`. Future provider events use the installed trigger configuration.

A repeated apply without configuration changes creates no writes or revisions.
Automations omitted from the manifest follow the existing removal behavior.
Shared automations must retain the same configuration in every existing stage,
or be removed from every stage.

Apply the ownership database migration first. Deploy the updated node-request
workers before the API so workers can process the new Datadog provisioning hook.
No new runtime service or configuration key is required. Existing node-request
and webhook workers process provisioning and cleanup.

Repository configuration reuses existing records and requires no additional
database migration or runtime configuration key.
