---
name: superplane-cli
description: Use when working with the SuperPlane CLI to discover available integrations, components, and triggers, and to build or troubleshoot canvases that connect trigger->component flows. Covers list/get command usage, interpreting configuration schemas, wiring channels between nodes, and resolving integration binding issues such as "integration is required".
---

# SuperPlane CLI Canvas Workflow

Use this workflow to build or debug canvases from the CLI.

## Discover what exists

Run these first:

```bash
superplane index integrations
superplane integrations list
superplane index triggers
superplane index components
```

Narrow to one integration:

```bash
superplane index triggers --from github
superplane index components --from github
superplane index components --from semaphore
```

Use `superplane integrations list` to list organization-connected integration instances (not just available providers).

Inspect required config fields and example payloads:

```bash
superplane index integrations --name github
superplane index triggers --name github.onPush
superplane index components --name semaphore.runWorkflow
superplane index components --name github.runWorkflow
superplane index components --name approval
```

List runtime options for `integration-resource` fields:

```bash
superplane integrations list-resources --id <integration-id> --type <type> --parameters key1=value1,key2=value2
```

Use `superplane integrations list` first to find valid integration IDs.

## Build canvas incrementally

Create a blank canvas first:

```bash
superplane canvases create <name>
superplane canvases get <name>
```

Edit a canvas file and update via:

```bash
superplane canvases update --file <canvas-file.yaml>
```

Use this resource header:

```yaml
apiVersion: v1
kind: Canvas
metadata:
  id: <canvas-id>
  name: <canvas-name>
spec:
  nodes: []
  edges: []
```

## Canvas YAML structure

Use this as the canonical shape when editing a canvas file.

Top-level fields:

- `apiVersion`: always `v1`
- `kind`: always `Canvas`
- `metadata.id`: canvas UUID (required for update)
- `metadata.name`: canvas name
- `spec.nodes`: list of trigger/component nodes
- `spec.edges`: list of directed graph connections

Node structure:

- Common fields: `id`, `name`, `type`, `configuration`, `position`, `paused`, `isCollapsed`
- Keep node `name` values unique within a canvas. Duplicate names can produce warnings and make expressions/diagnostics ambiguous.
- `type` must be `TYPE_TRIGGER` or `TYPE_COMPONENT`
- Trigger nodes must include `trigger.name`
- Component nodes must include `component.name`
- Integration-backed nodes should include `integration.id` (`integration.name` can be empty string)
- `errorMessage` and `warningMessage` are optional but useful for troubleshooting
- `metadata` is optional and usually server-populated

Edge structure:

- `sourceId`: upstream node id
- `targetId`: downstream node id
- `channel`: output channel from source node (`default`, `passed`, `approved`, etc.)

Minimal example:

```yaml
apiVersion: v1
kind: Canvas
metadata:
  id: <canvas-id>
  name: <canvas-name>
spec:
  nodes:
    - id: trigger-main
      name: github.onPush
      type: TYPE_TRIGGER
      trigger:
        name: github.onPush
      integration:
        id: <github-integration-id>
        name: ""
      configuration:
        repository: owner/repo
        refs:
          - type: equals
            value: refs/heads/main
      position:
        x: 120
        y: 100
      paused: false
      isCollapsed: false

    - id: component-ci
      name: semaphore.runWorkflow
      type: TYPE_COMPONENT
      component:
        name: semaphore.runWorkflow
      integration:
        id: <semaphore-integration-id>
        name: ""
      configuration:
        project: <project-id-or-name>
        pipelineFile: .semaphore/semaphore.yml
        ref: refs/heads/main
      position:
        x: 480
        y: 100
      paused: false
      isCollapsed: false

  edges:
    - sourceId: trigger-main
      targetId: component-ci
      channel: default
```

## Node and edge wiring rules

Use `TYPE_TRIGGER` for trigger nodes and `TYPE_COMPONENT` for component nodes.

For triggers, set:

- `trigger.name` to the trigger id (example: `github.onPush`)

For components, set:

- `component.name` to the component id (example: `semaphore.runWorkflow`)

For graph flow, set edges:

- `sourceId` and `targetId` for connection
- `channel` when routing specific outputs (example: `passed`, `approved`)

Typical gated flow:

1. Trigger -> CI component
2. CI `passed` -> `approval`
3. `approval` `approved` -> deploy component

## Configure integration-backed fields correctly

When a field type is `integration-resource` (such as `repository` or `project`), the org must have a configured integration instance for that provider.

Symptoms of missing binding:

- Node `errorMessage` contains `integration is required`

How to resolve:

1. Run `superplane integrations list` and confirm required providers are connected for the org.
2. Use `superplane integrations get <integration-id>` to inspect one connected integration when needed.
3. Ensure the provider integration (GitHub, Semaphore, etc.) is installed and authenticated for the organization.
4. Reopen the node config and select valid provider resources for required fields.
5. Use `superplane integrations list-resources --id <integration-id> --type <type> --parameters ...` to inspect valid option IDs/names.
6. Re-run `superplane canvases get <name>` and confirm node errors are cleared.

## Troubleshooting checklist

Run this after every update:

```bash
superplane canvases get <name>
```

Check:

- All required `configuration` fields are present.
- Edges use the correct output channels.
- No node `errorMessage` remains.
- No node `warningMessage` indicates duplicate names (for example: `Multiple components named "semaphore.runWorkflow"`).
- Expressions reference existing node names.
