# Manifest Generation Rules

## Purpose

This document defines how the SDK turns authoring-time extension objects into the serialized manifest consumed by the engine.

This is the bridge between:

- the authoring model:
  - `defineExtension({ metadata, integrations, components, triggers })`
  - one file per integration/component/trigger
- the serialized manifest model:
  - static JSON-compatible metadata used for discovery, installation, validation, and UI

## Design Principles

- The serialized manifest contains static metadata only.
- The SDK serializes authoring-time static data during packaging or discovery.
- Runtime behavior is not encoded into the serialized manifest.
- Default values are materialized into the serialized manifest where appropriate.
- Broken references or invalid block relationships must fail packaging.

## Input Shape

The SDK starts from:

```ts
defineExtension({
  metadata,
  runtime,
  integrations,
  components,
  triggers,
});
```

Where:

- `metadata` is required
- `runtime` is optional
- `integrations`, `components`, and `triggers` default to empty arrays when omitted

## Output Shape

The output is a serialized `ManifestV1`:

```ts
{
  apiVersion: "spx/v1",
  kind: "extension",
  metadata: ...,
  runtime: ...,
  integrations: [...],
  components: [...],
  triggers: [...],
}
```

## Top-Level Rules

### `apiVersion`

Always:

```yaml
apiVersion: spx/v1
```

### `kind`

Always:

```yaml
kind: extension
```

### `metadata`

Copied as-is from the authoring definition.

### `runtime`

If omitted by the author, default to:

```yaml
runtime:
  profile: portable-web-v1
```

## Integration Serialization Rules

For each `IntegrationDefinition`:

- `name` is copied as-is
- `label` is copied as-is
- `icon` is copied as-is
- `description` is copied as-is
- `instructions` is copied as-is when present
- `configuration` is copied as-is and serialized
- `actions` defaults to `[]` when omitted
- `resourceTypes` defaults to `[]` when omitted

Example:

Authoring:

```ts
export const cloudflare = {
  name: "cloudflare",
  label: "Cloudflare",
  icon: "cloud",
  description: "Manage Cloudflare zones, rules, and DNS",
  configuration: [],
} satisfies IntegrationDefinition;
```

Serialized:

```yaml
- name: cloudflare
  label: Cloudflare
  icon: cloud
  description: Manage Cloudflare zones, rules, and DNS
  configuration: []
  actions: []
  resourceTypes: []
```

## Component Serialization Rules

For each `ComponentDefinition`:

- `name` is copied as-is
- `integration` is copied as-is when present
- `label` is copied as-is
- `description` is copied as-is
- `icon` is copied as-is
- `color` is copied as-is
- `configuration` is copied as-is and serialized
- `actions` defaults to `[]` when omitted
- `outputChannels` defaults to:

```yaml
outputChannels:
  - name: default
    label: Default
```

when omitted

Example:

Authoring:

```ts
export const createDnsRecord = {
  name: "cloudflare.createDnsRecord",
  integration: "cloudflare",
  label: "Create DNS Record",
  description: "Create a DNS record in a Cloudflare zone",
  icon: "cloud",
  color: "orange",
  configuration: [],
  async execute({ runtime }) {
    runtime.executionState.pass();
  },
} satisfies ComponentDefinition;
```

Serialized:

```yaml
- name: cloudflare.createDnsRecord
  integration: cloudflare
  label: Create DNS Record
  description: Create a DNS record in a Cloudflare zone
  icon: cloud
  color: orange
  configuration: []
  actions: []
  outputChannels:
    - name: default
      label: Default
```

## Trigger Serialization Rules

For each `TriggerDefinition`:

- `name` is copied as-is
- `integration` is copied as-is when present
- `label` is copied as-is
- `description` is copied as-is
- `icon` is copied as-is
- `color` is copied as-is
- `configuration` is copied as-is and serialized
- `actions` defaults to `[]` when omitted

## Static Data Rules

Rules:

- manifest metadata must be JSON-serializable
- static arrays and values must not rely on mutable runtime state
- packaging should reject invalid data shapes

## Cross-Reference Validation Rules

Packaging must fail if any of these rules are violated:

- block names must be unique across integrations, components, and triggers within the extension
- a component or trigger using an `integration-resource` field must declare `integration`

Packaging should allow these cases:

- a component `integration` may reference an integration defined by the same extension
- a component `integration` may reference an integration provided by another extension
- a trigger `integration` may reference an integration defined by the same extension
- a trigger `integration` may reference an integration provided by another extension

Engine/runtime note:

- validation that an `integration` reference resolves to an installed integration may happen at install time or runtime, not package time

## Serialization Defaults Summary

- `runtime.profile` defaults to `portable-web-v1`
- `integrations` defaults to `[]`
- `components` defaults to `[]`
- `triggers` defaults to `[]`
- `actions` defaults to `[]`
- `resourceTypes` defaults to `[]`
- `outputChannels` defaults to `[{ name: "default", label: "Default" }]`

SDK note:

- the SDK should expose a shared `DEFAULT_OUTPUT_CHANNEL` constant with that value

## Non-Goals

These rules do not define:

- runtime dispatch
- execution ordering
- permission enforcement
- sandbox behavior
- transport protocol

Those belong to separate runtime and engine contracts.
