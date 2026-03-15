# Manifest Schema Draft

## Purpose

This document defines the first draft of the static manifest schema for extensions.

The manifest is the declarative, discoverable surface of an extension. It describes:

- which blocks the extension provides
- how those blocks appear in the UI
- which configuration fields they expose
- which actions, resource types, and channels they declare
- which runtime handlers must exist in the implementation

The manifest does not embed implementation logic or runtime state.

This document describes the serialized static manifest. It is not the authoring format used by the SDK. The SDK should derive this manifest from exported integration, component, and trigger objects.

## Top-Level Shape

```yaml
apiVersion: spx/v1
kind: extension
metadata:
  id: cloudflare
  name: Cloudflare Extension
  version: 0.1.0
  description: Cloudflare integration, components, and triggers
runtime:
  profile: portable-web-v1
integrations: []
components: []
triggers: []
```

## Design Rules

- `integrations`, `components`, and `triggers` are top-level blocks.
- Components and triggers may reference an integration through `integration`.
- Integrations may reference which component and trigger names they expose for grouping and discovery.
- The manifest should stay language-neutral.
- Runtime wiring is derived by the SDK and is not authored directly.

## Top-Level Fields

### `apiVersion`

Current value:

```yaml
apiVersion: spx/v1
```

### `kind`

Current value:

```yaml
kind: extension
```

### `metadata`

```yaml
metadata:
  id: cloudflare
  name: Cloudflare Extension
  version: 0.1.0
  description: Cloudflare integration, components, and triggers
```

Fields:

- `id`: stable extension identifier
- `name`: human-readable extension name
- `version`: immutable extension version
- `description`: optional summary

### `runtime`

```yaml
runtime:
  profile: portable-web-v1
```

Fields:

- `profile`: declared runtime compatibility profile

## Integration Block

An integration block is the declarative form of the current integration contract.

```yaml
integrations:
  - name: cloudflare
    label: Cloudflare
    icon: cloud
    description: Manage Cloudflare zones, rules, and DNS
    instructions: |
      ## Create a Cloudflare API Token
      ...
    configuration: []
    actions: []
    resourceTypes:
      - zone
      - dns_record
      - redirect_rule
```

Fields:

- `name`: stable programmatic name and block identifier
- `label`: UI display label
- `icon`: UI icon identifier
- `description`: short description
- `instructions`: markdown instructions for connection setup
- `configuration`: list of fields shown when configuring the integration
- `actions`: static action definitions exposed by the integration
- `resourceTypes`: resource types that can be listed through `listResources`

## Component Block

A component block is the declarative form of the current component contract.

```yaml
components:
  - name: cloudflare.createDnsRecord
    integration: cloudflare
    label: Create DNS Record
    description: Create a DNS record in a Cloudflare zone
    icon: cloud
    color: orange
    outputChannels:
      - name: default
        label: Default
    configuration: []
    actions: []
```

Fields:

- `name`: stable node registration name and block identifier
- `integration`: optional integration name reference; this may refer to an integration defined by the same extension or by another installed extension
- `label`: UI display label
- `description`: short description
- `icon`: UI icon identifier
- `color`: UI color token
- `outputChannels`: declared output channels
- `configuration`: configuration fields shown to users
- `actions`: static action definitions

Defaulting:

- if the authoring definition omits `outputChannels`, the serialized manifest should contain:

```yaml
outputChannels:
  - name: default
    label: Default
```

- if the authoring definition omits `actions`, the serialized manifest should contain `actions: []`

## Trigger Block

A trigger block is the declarative form of the current trigger contract.

```yaml
triggers:
  - name: github.issueOpened
    integration: github
    label: Issue Opened
    description: Emits when a GitHub issue is opened
    icon: github
    color: black
    configuration: []
    actions: []
```

Fields:

- `name`: stable node registration name and block identifier
- `integration`: optional integration name reference; this may refer to an integration defined by the same extension or by another installed extension
- `label`: UI display label
- `description`: short description
- `icon`: UI icon identifier
- `color`: UI color token
- `configuration`: configuration fields shown to users
- `actions`: static action definitions

Defaulting:

- if the authoring definition omits `actions`, the serialized manifest should contain `actions: []`

## Shared Substructures

### Configuration Field

This is a manifest-level, language-neutral form of the existing `configuration.Field`.

The schema is a discriminated union keyed by `type`.

Shared fields across all configuration fields:

- `name`
- `label`
- `type`
- `placeholder`
- `description`
- `required`
- `default`
- `togglable`
- `disallowExpression`
- `sensitive`
- `typeOptions`
- `visibilityConditions`
- `requiredConditions`
- `validationRules`

Supported field types currently mirror the engine field surface:

- `string`
- `text`
- `expression`
- `xml`
- `number`
- `boolean`
- `select`
- `multi-select`
- `list`
- `object`
- `time`
- `date`
- `datetime`
- `timezone`
- `days-of-week`
- `time-range`
- `day-in-year`
- `cron`
- `user`
- `role`
- `group`
- `integration-resource`
- `any-predicate-list`
- `git-ref`
- `secret-key`

#### String Field

```yaml
- name: apiToken
  label: API Token
  type: string
  required: true
  sensitive: true
  description: API token
  typeOptions:
    string:
      minLength: 20
      maxLength: 128
```

#### Number Field

```yaml
- name: ttl
  label: TTL
  type: number
  description: TTL in seconds
  typeOptions:
    number:
      min: 1
      max: 86400
```

#### Boolean Field

```yaml
- name: proxied
  label: Proxied
  type: boolean
  default: false
```

#### Select Field

```yaml
- name: type
  label: Type
  type: select
  required: true
  typeOptions:
    select:
      options:
        - label: A
          value: A
        - label: AAAA
          value: AAAA
```

#### Multi-Select Field

```yaml
- name: environments
  label: Environments
  type: multi-select
  typeOptions:
    multiSelect:
      options:
        - label: Production
          value: prod
        - label: Staging
          value: staging
```

#### Integration Resource Field

```yaml
- name: repository
  label: Repository
  type: integration-resource
  required: true
  typeOptions:
    resource:
      type: repository
      useNameAsValue: true
```

#### Any Predicate List Field

```yaml
- name: refs
  label: Refs
  type: any-predicate-list
  required: true
  default:
    - type: equals
      value: refs/heads/main
  typeOptions:
    anyPredicateList:
      operators:
        - label: Equals
          value: equals
        - label: Not Equals
          value: notEquals
        - label: Matches
          value: matches
```

Additional notes:

- `integration-resource` fields use `typeOptions.resource`
- `select` fields use `typeOptions.select`
- `multi-select` fields use `typeOptions.multiSelect`
- `any-predicate-list` fields use `typeOptions.anyPredicateList`
- `list` fields use `typeOptions.list`
- `object` fields use `typeOptions.object`

#### Visibility Conditions

Visibility conditions are also typed.

Equality:

```yaml
visibilityConditions:
  - field: provider
    value: aws
```

Membership:

```yaml
visibilityConditions:
  - field: type
    operator: in
    values: [A, AAAA, CNAME]
```

Presence:

```yaml
visibilityConditions:
  - field: apiToken
    values: [configured]
```

### Action

```yaml
- name: refresh
  description: Refresh remote metadata
  userAccessible: false
  parameters: []
```

### Output Channel

```yaml
- name: default
  label: Default
  description: Default output channel
```

## Validation Rules

- All block names must be unique within the extension.
- All action names must be unique within a block.
- All output channel names must be unique within a component.
- Configuration field names must be unique within a block.
- `name` should be globally unique within its block type.
- A field's `default` must match the field's declared `type`.
- Field-specific `typeOptions` must match the field's declared `type`.
- `sensitive` is only valid on `string` fields.
- `integration-resource` fields must declare `typeOptions.resource.type`.
- blocks using `integration-resource` fields should declare `integration`.
- `select` and `multi-select` fields must declare at least one option.
- Visibility conditions must reference fields that exist in the same block.
- Secret user input should use `type: string` with `sensitive: true`.

## Cloudflare Example

```yaml
apiVersion: spx/v1
kind: extension
metadata:
  id: cloudflare
  name: Cloudflare Extension
  version: 0.1.0
  description: Manage Cloudflare zones, DNS records, and rules
runtime:
  profile: portable-web-v1
integrations:
  - name: cloudflare
    label: Cloudflare
    icon: cloud
    description: Manage Cloudflare zones, rules, and DNS
    instructions: |
      ## Create a Cloudflare API Token
      ...
    configuration:
      - name: apiToken
        label: API Token
        type: string
        required: true
        sensitive: true
        description: Cloudflare API Token with appropriate permissions
        typeOptions:
          string:
            minLength: 20
    actions: []
    resourceTypes:
      - zone
      - redirect_rule
      - dns_record
components:
  - name: cloudflare.createDnsRecord
    integration: cloudflare
    label: Create DNS Record
    description: Create a DNS record in a Cloudflare zone
    icon: cloud
    color: orange
    outputChannels:
      - name: default
        label: Default
    configuration:
      - name: zone
        label: Zone
        type: integration-resource
        required: true
        typeOptions:
          resource:
            type: zone
            useNameAsValue: true
      - name: type
        label: Type
        type: select
        required: true
        typeOptions:
          select:
            options:
              - label: A
                value: A
              - label: AAAA
                value: AAAA
              - label: CNAME
                value: CNAME
      - name: ttl
        label: TTL
        type: number
        required: false
        typeOptions:
          number:
            min: 1
            max: 86400
      - name: proxied
        label: Proxied
        type: boolean
        default: false
        visibilityConditions:
          - field: type
            values: [A, AAAA, CNAME]
    actions: []
triggers: []
```
