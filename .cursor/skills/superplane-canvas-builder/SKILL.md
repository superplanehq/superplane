---
name: superplane-cli-canvas
description: Use when working with the SuperPlane CLI to discover available integrations, components, and triggers, and to build or troubleshoot canvases that connect trigger->component flows. Covers list/get command usage, interpreting configuration schemas, wiring channels between nodes, and resolving integration binding issues such as "integration is required".
---

# SuperPlane CLI Canvas Workflow

Use this workflow to build or debug canvases from the CLI.

## Discover what exists

Run these first:

```bash
superplane list integrations
superplane list integrations --connected
superplane list triggers
superplane list components
```

Narrow to one integration:

```bash
superplane list triggers --from github
superplane list components --from github
superplane list components --from semaphore
```

Use `--connected` to list organization-connected integration instances (not just available providers). The CLI resolves this via `/api/v1/me` to get `organizationId`, then `/api/v1/organizations/{organizationId}/integrations`.

Inspect required config fields and example payloads:

```bash
superplane get trigger --name github.onPush
superplane get component --name semaphore.runWorkflow
superplane get component --name github.runWorkflow
superplane get component --name approval
```

## Build canvas incrementally

Create a blank canvas first:

```bash
superplane create canvas <name>
superplane get canvas <name>
```

Edit a canvas file and update via:

```bash
superplane update -f <canvas-file.yaml>
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

1. Run `superplane list integrations --connected` and confirm required providers are connected for the org.
2. Ensure the provider integration (GitHub, Semaphore, etc.) is installed and authenticated for the organization.
3. Reopen the node config and select valid provider resources for required fields.
4. Re-run `superplane get canvas <name>` and confirm node errors are cleared.

## Troubleshooting checklist

Run this after every update:

```bash
superplane get canvas <name>
```

Check:

- All required `configuration` fields are present.
- Edges use the correct output channels.
- No node `errorMessage` remains.
- Expressions reference existing node names.
