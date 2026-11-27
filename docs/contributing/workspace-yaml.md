# Workspace YAML

Workspace YAML configures the workspace display name, existing stages, and
automations. CLI export, plan, and apply use the same v2 Factory contract.
The workspace ID or key in `--workspace` selects the destination. The manifest
`name` sets its display name. A rename preserves the workspace key and ID.
Multiple workspaces can use the same display name.

## Export, plan, and apply

Export a workspace before you edit its configuration:

```sh
superplane workspaces export --workspace WORKSPACE
superplane workspaces plan --workspace WORKSPACE
superplane workspaces apply --workspace WORKSPACE
```

Export writes `.superplane/main.yml` and referenced custom automation files.
It refuses to replace existing files unless you pass `--overwrite`.
Plan displays configuration changes without applying them. Apply displays a
plan and requests confirmation. Pass `--yes` for noninteractive use.
The captured plan includes a base-state token. Apply rejects concurrent
configuration changes so it cannot overwrite an outdated workspace state.

A manifest can use any `.yml` or `.yaml` filename:

```sh
superplane workspaces plan --workspace WORKSPACE -f workspace.yml
superplane workspaces apply --workspace WORKSPACE -f workspace.yml
```

Without `--root`, local references must remain inside the manifest directory.
Use `--root` to share automation sources across manifest directories:

```sh
superplane workspaces plan --workspace WORKSPACE --root . -f .superplane/workspaces/workspace-1.yml
superplane workspaces apply --workspace WORKSPACE --root . -f .superplane/workspaces/workspace-1.yml
```

This manifest can reference `../automations/review.yml`. Paths resolve relative
to the selected manifest. They must remain inside the explicit root.
The CLI packages only the selected manifest and referenced files. Several
references to the same normalized path share one bundle entry. Other manifests
and unrelated repository files stay outside the bundle.

Absolute references, root escapes, backslashes, non-YAML references, and
self-references fail validation. Manifest files, references, and their parent
directories cannot use symbolic links. Input files must be regular files.
Stdin (`-f -`) supports manifests without local automation references.

## Factory manifest

```yaml
apiVersion: v2
kind: Factory
name: Application workspace
stages:
  - id: backlog
    name: Backlog
    automations: []
  - id: implement
    name: Implement
    run: implement-changes
    automations:
      - id: implement-changes
        uses: '@superplane/implement'
        with:
          agent:
            additional_instructions: Keep changes focused.
            thinking_level: high
            timeout: 2h
  - id: verify
    name: Verify
    automations:
      - id: review-changes
        uses: ../automations/review.yml
  - id: done
    name: Done
    automations: []
```

Use IDs from a fresh export when you configure an existing workspace.
Stage IDs are unique within the workspace. Automation IDs are unique across
all stages. Stage and automation IDs use separate namespaces. Two instances
of one built-in template need different automation IDs.

On first export, SuperPlane assigns and saves missing public IDs for the
selected line's stages and workspace automations. Existing workspaces keep
their data until export. Later exports reuse saved IDs. Display-name changes, manifest
moves, and source-file moves preserve IDs and existing destination records.
Copying a manifest into another workspace creates independent records with the
same public IDs. The target must have the required existing stages.
Changing an ID means removal and addition, subject to required-resource rules.
There is no YAML ID-rename operation.

A custom automation entry can omit `name`. SuperPlane reads its display name
from the referenced definition. An explicit manifest name is also supported.
Conflicting names for one shared identity fail validation. A stage `run` refers
to an automation ID declared in the workspace. Multiple phases can reference
one automation declaration.
Stage concurrency stays in workspace settings. Omission preserves the live
limit, including non-default limits. Do not include `max_parallelism`.

## Custom automation definition

```yaml
apiVersion: v2
kind: Automation
name: Review changes
timeout: 90m
agent:
  skip_setup_commands: false
  skip_repository_checkout: false
triggers:
  - uses: manual
steps:
  - name: Review
    prompt: Review the changes and report defects.
```

Omit `agent` to inherit the workspace agent. A model-only selection or execution
options can also inherit workspace credentials. Explicit model, credential,
instruction, and setup choices remain overrides even when they match current
workspace defaults. Custom setup and checkout options belong under `agent`.
Generated defaults, including the `sparkles` icon, are omitted from exports.
An explicit non-default icon remains in the definition.

Use `timeout` for a duration such as `2h`, `90m`, or `30s`. Durations resolve to
whole seconds and follow existing execution limits and zero/default behavior.
Negative, invalid, fractional-second, and overflowing durations fail validation.
The custom definition timeout controls the automation. Built-in
`with.agent.timeout` controls the logical agent execution branches.

Triggers use `uses`. Schedule configuration uses a fixed offset string:

```yaml
triggers:
  - uses: schedule
    config:
      cron: '0 9 * * *'
      timezone: '1'
  - uses: manual
run_context: workspace
```

`timezone` preserves the existing fixed UTC offset convention. Negative and
fractional offsets are supported. IANA zones and daylight-saving rules are
outside this contract.

Task-event triggers, manual-only automations, and phase runners run with task
context by default. Schedule-plus-manual automations remain task-free. Use
`run_context: workspace` for a manual-only automation without a task.
Use `run_context: task` or `run_context: workspace` when triggers do not determine
the context.
Webhooks can run in every board stage. When `run_context` is absent, webhooks
can share a definition with manual or task triggers. Each trigger keeps its
context. Task events and schedules cannot share one definition.
Incompatible trigger and context combinations fail before publication.

## Repository information

The optional `repositories` section describes destination repository settings.
It cannot change repositories through YAML. Configure destination repositories
through workspace settings before applying a copied manifest.
Keep matching repository information or omit this section. Manual repository
changes can make exported YAML stale. Refresh the YAML from export or remove
the optional section before the next apply.

Supported repository bindings, intake settings, template settings, and grant
validation remain available. Built-in agent settings use the singular
`with.agent` mapping. Ordinary options do not expose internal canvas node IDs.
Organization built-in reset preserves supported explicit settings.

## Replace older YAML

Older YAML syntax and exported identifiers are not accepted. Replace old files
with a fresh export, or rewrite them to this contract. There are no legacy
aliases or conversion commands.

Use stage `id`, trigger `uses`, schedule `config.timezone`, duration `timeout`,
and explicit `run_context` when needed. Move custom setup and checkout options
under `agent`. Move built-in instructions, thinking level, and timeout under
`with.agent`. Removed fields such as stage `key`, `max_parallelism`, trigger
`use`, `utc_offset`, `runs_on_task`, and `with.agents` fail validation.

Apply the schema migration before deploying the updated API and workers.
The migration does not assign IDs or rewrite existing stages. Export saves
missing IDs in one transaction. A failed export does not save partial changes.
Export the destination before planning or applying YAML that manages existing resources.
Existing records and execution settings remain in place.
GitHub synchronization and stage creation or removal are separate work.
