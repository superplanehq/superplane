# Factory-as-Code: All stages and automations

This experimental POC exports, plans, and applies configuration for every
existing stage on the selected workspace line. It includes Backlog, every
phase, custom stages, Verify, and Done. It includes their built-in and custom
automations. Export reads current published configuration, including UI edits.

The POC supports existing workspaces. It does not export every workspace
setting and is not a workspace backup. Repository connections, credentials,
workspace agent defaults, setup commands, skills, tasks, and run history stay
outside this bundle.

Enable the `factory_as_code` experimental feature for the organization through
the existing admin feature controls. The organization must also have the
`factories` feature. Export requires `factories:read`. Plan and apply require
`factories:update`.

Changing stage names or order requires `factory_custom_stages`. Adding or changing a
custom automation definition requires its existing feature: `factory_custom_stages`
for custom-stage automations, or `factory_custom_automations` for other custom
automations. An unchanged export remains valid when those features are disabled.

## Manifest and custom files

The bundle contains `.superplane/factory.yaml` and the custom automation files
that it references. Built-ins use `@superplane/` aliases. Custom files use the
existing Automation v2 language, with triggers and ordered Bash or Prompt steps.
The export does not contain Canvas YAML or credential values.

A manifest can contain stages such as:

```yaml
apiVersion: v2
kind: Factory
name: Example workspace
stages:
  - key: backlog
    name: Backlog
    kind: backlog
    automations: []
  - key: implementation
    name: Implement
    kind: phase
    run: implement
    max_parallelism: 10
    automations:
      - id: implement
        name: Implement
        uses: "@superplane/implement"
        with:
          additional_instructions: |
            Follow AGENTS.md.
  - key: custom-review
    name: Review changes
    kind: custom
    automations:
      - id: review
        name: Review changes
        uses: automations/review.yaml
  - key: verify
    name: Verify
    kind: verify
    automations:
      - id: merge-confidence
        name: Merge confidence
        uses: "@superplane/merge-confidence"
        with:
          agents:
            assess-risk:
              additional_instructions: Check database migration risks.
              thinking_level: low
              execution_timeout_seconds: 3600
              skip_setup_commands: false
              skip_repository_checkout: false
  - key: done
    name: Done
    kind: done
    automations: []
```

The example illustrates the structure. Export supplies the actual stage keys,
automation IDs, names, references, and settings for the workspace. Keep existing
automation IDs and bindings. Custom stage keys can be opaque values.

This reference resolves relative to the manifest directory:

```text
.superplane/factory.yaml
.superplane/automations/review.yaml
```

For example, `automations/review.yaml` contains:

```yaml
apiVersion: v2
kind: Automation
name: Review changes
triggers:
  - use: manual
steps:
  - name: Review
    prompt: Review the changes and report problems.
```

The CLI reads the complete bundle once before plan. Apply submits that captured
bundle after confirmation. Absolute paths, traversal, and symbolic links in
custom file paths are rejected. Standard input supports manifests without
custom file references. Use `-f FILE` when a manifest references custom files.

## Supported changes

The POC can update existing stage names and stage order, subject to the current
board placement rules. It can update phase concurrency through positive
`max_parallelism` values. The `run` field keeps the phase bound to its existing
execution automation.

Built-ins remain template references. Their `with` values expose a bounded set
of supported agent settings. Unexported built-in settings keep their live
values. Export includes built-ins even when they have no editable settings.
`additional_instructions: ""` clears user instructions without changing the
managed prompt. Instructions preserve their whitespace.

Other built-in agents use `with.agents.<node-id>`, as the Merge confidence
example shows. These keys are stable template node IDs, such as `assess-risk`.
Keep the node IDs from the export. They are not Canvas database IDs.

The following agent settings are supported:

| Field | Value |
| --- | --- |
| `additional_instructions` | String; an empty string clears the instructions |
| `model` | String supported by the existing agent |
| `thinking_level` | String supported by the existing agent |
| `execution_timeout_seconds` | Integer from 0 through 86400 |
| `skip_setup_commands` | Boolean |
| `skip_repository_checkout` | Boolean |

Implement uses `with.additional_instructions` for its implementation agent.
Keep this required field in the manifest. Its other settings use the nested
agent mapping. Other omitted settings keep their current values. Explicit empty
strings, false values, and zero timeouts are preserved where the existing agent
supports them.

The current built-in aliases are:

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

Intake aliases use `@superplane/intake/` followed by `github-issues`,
`dependabot-alerts`, `jira-issues`, `sentry-exceptions`, `pagerduty-incidents`,
`productive-tasks`, `datadog`, or `linear-issues`.

Custom files preserve triggers, step order, step content, supported execution
options, named secrets, and agent overrides. Phase runners and custom-stage
automations use one manual trigger, compiled to the existing On Run dispatch
payload. Ordinary Backlog, Implement, Verify, and Done automations keep their
current trigger rules. Custom-stage automations continue to run together.

To add a custom automation, create its Automation v2 file and add an entry to
the stage's `automations` list:

```yaml
      - id: review-changes
        name: Review changes
        uses: automations/review.yaml
```

Use a unique ID in that stage and a unique name in the workspace. The manifest's
`name` sets the automation display name. Apply also saves that name in the custom
Automation document. New custom automations can use
Backlog, Implement, Verify, Done, or an existing custom stage.

To rename an existing automation, change its `name` in the manifest. Keep the
ID and binding from export. Custom files do not need a second name edit.
A name-only change preserves the automation identity, graph revision, runs,
and stage bindings. Names must be unique within the workspace. Shared automation
entries must request the same name in every stage. Built-in names survive admin
reset.

To remove an optional automation, remove its manifest entry. An empty stage
uses `automations: []`. Deleting an automation file alone reports a missing
file diagnostic. Apply removes associated intake or pull request handler
records and the saved custom source when present. It preserves run history.
Task analysis and phase runners cannot be removed. A shared automation must
be removed from every covered stage together. Workspace-wide removals affect
every line that shares the automation.

Plan displays additions and removals before apply requests confirmation.
Repeating apply with the same files reports no changes. Export assigns canonical
IDs and paths to new custom automations. Apply also retains their original
manifest IDs, so they can be renamed before another export. Duplicate built-ins
use stable ID suffixes, so name changes do not change their identities.

This POC does not create or remove stages, move automations between stages,
install new built-in aliases, or change phase bindings. Install built-ins
through the existing workspace controls before exporting them.
It does not create workspaces or synchronize Git. Additional workspace lines
and unrelated resources remain unchanged. Workspace-wide automations retain
their existing sharing across lines.

An unsupported graph causes export to fail with the automation name. The
export must not silently omit an automation or emit raw Canvas YAML.

## API contracts

All routes use the existing authenticated API and organization headers.
`factory_id` accepts a workspace key or UUID. The CLI needs only `--workspace`.
The API first selects the valid onboarding line. If that line is unavailable,
it selects the sole existing line. Otherwise, the target is ambiguous and the
operation fails. API clients can select a line explicitly with `line_id`.
The CLI does not expose a line flag.

| Operation | Route | Request | Response |
| --- | --- | --- | --- |
| Export | `GET /api/v1/factories/{factory_id}/configuration` | Path and query parameters | `bundle`, `coverage` |
| Plan | `POST /api/v1/factories/{factory_id}/configuration:plan` | `bundle` | `changes`, `diagnostics`, `baseStateToken`, `coverage` |
| Apply | `POST /api/v1/factories/{factory_id}/configuration:apply` | `bundle`, `baseStateToken` | `changed`, `changes`, `coverage` |

Coverage is `All stages and automations`. This describes stage coverage, not a
complete workspace export. JSON uses the generated API's camelCase fields.

```json
{
  "manifestPath": ".superplane/factory.yaml",
  "files": [
    {"path": ".superplane/factory.yaml", "content": "<Factory YAML>"},
    {"path": ".superplane/automations/review.yaml", "content": "<Automation v2 YAML>"}
  ]
}
```

Each change identifies its resource, field path, previous value, and requested
value. Diagnostics identify the file, field, and problem. Invalid input returns
plan diagnostics and no token. Apply rejects invalid input before any write.

The base-state token binds the selected workspace, line configuration, all
covered automation revisions, and the normalized desired configuration. Apply
locks the affected resources, reloads their state, and verifies the token.
Request a new plan after any intervening configuration save.

A valid apply publishes only changed automations and saves the stage changes
in one database transaction. Notifications occur after commit. Invalid input,
stale plans, and publication failures leave persisted configuration unchanged.
A repeated apply creates no automation revision. YAML formatting alone does
not change configuration. Apply does not replay intake imports.

## Local demonstration

Complete normal workspace onboarding and enable the experimental feature.
Build the CLI through the repository target:

```bash
make cli.build.binary OS=linux ARCH=amd64
```

Configure the local API URL, organization, and token through the existing CLI
authentication flow. Select the workspace and an export directory:

```bash
FACTORY_WORKSPACE=your-workspace-key
FACTORY_DEMO_DIR=/tmp/superplane-factory-as-code-demo
mkdir -p "$FACTORY_DEMO_DIR"
git -C "$FACTORY_DEMO_DIR" init
build/cli workspace export --workspace "$FACTORY_WORKSPACE" \
  --output-dir "$FACTORY_DEMO_DIR"
git -C "$FACTORY_DEMO_DIR" add .superplane
```

Inspect the manifest and referenced files. Confirm that every board stage and
its automations appear. Edit a supported built-in setting or a custom step.
Then inspect and apply the plan:

```bash
build/cli workspace plan --workspace "$FACTORY_WORKSPACE" \
  -f "$FACTORY_DEMO_DIR/.superplane/factory.yaml"
build/cli workspace apply --workspace "$FACTORY_WORKSPACE" \
  -f "$FACTORY_DEMO_DIR/.superplane/factory.yaml"
```

Apply displays a fresh plan and requests confirmation. Use `--yes` for
noninteractive use. An unchanged plan skips apply.

Repeat apply and confirm that it reports no changes. Save a supported change
in the UI, export again, and inspect the files:

```bash
build/cli workspace export --workspace "$FACTORY_WORKSPACE" \
  --output-dir "$FACTORY_DEMO_DIR" --overwrite
git -C "$FACTORY_DEMO_DIR" diff -- .superplane
```

Export checks every bundle file and destination before replacement. It stages
all content first, replaces each file atomically, and writes the manifest last.
It refuses existing destinations without `--overwrite` and preserves unrelated
files. It does not delete custom files left by an older export.

## Implementation and verification

The backend owns discovery, built-in settings validation, custom compilation,
comparison, stale-state protection, and transactional publication. The CLI owns
file collection, safe export paths, plan display, and confirmation.

Use the targeted checks before the required Go checks:

```bash
make test PKG_TEST_PACKAGES='./pkg/grpc/actions/factories ./pkg/cli/commands/workspaces ./pkg/automationspec ./pkg/authorization ./pkg/features'
make format.go
make lint
make check.build.app
make cli.build.binary OS=linux ARCH=amd64
make pb.gen
make check.proto.field.numbers
```

Generated protobufs and SDKs stay ignored. This POC adds no runtime service,
worker, deployment configuration, or Git webhook processing.
