# Integrations and Secrets

## Overview

Integrations connect SuperPlane Apps to external systems, while secrets protect
the credentials and values those connections and workflow nodes need. Builders
must be able to discover capabilities and bind configured integrations without
exposing secret material.

## Terminology

- **Integration:** An organization-scoped connection that provides components,
  triggers, resources, and credentials.
- **Secret:** A named, access-controlled collection of sensitive key-value
  data.
- **Capability:** A trigger or component behavior made available by an
  integration.

## Requirements

### REQ-INT-001: Discover integration capabilities

**User story:** As a workflow builder, I want to browse available integrations,
components, triggers, and resources, so that I can choose a supported action
for my workflow.

**Acceptance criteria:**

- **AC-INT-001.1:** When a builder browses the catalog, SuperPlane shall
  identify available capabilities and the integration each capability needs.
- **AC-INT-001.2:** When a capability is unavailable for the organization,
  SuperPlane shall not present it as ready to run.

### REQ-INT-002: Configure an integration

**User story:** As an organization administrator, I want guided integration
setup and validation, so that workflow builders can use an authorized
connection.

**Acceptance criteria:**

- **AC-INT-002.1:** When an administrator completes all required setup steps
  with valid values, SuperPlane shall show the integration as usable.
- **AC-INT-002.2:** When setup validation fails, SuperPlane shall identify the
  affected step and shall not represent the integration as ready.

### REQ-INT-003: Bind Apps to integrations

**User story:** As a workflow builder, I want to select the appropriate
configured integration for a node, so that its trigger or component acts in
the intended account and scope.

**Acceptance criteria:**

- **AC-INT-003.1:** When multiple compatible integrations exist, SuperPlane
  shall let the builder distinguish and select the intended binding.
- **AC-INT-003.2:** When a required binding is missing or deleted, SuperPlane
  shall prevent successful execution and explain the missing dependency.

### REQ-INT-004: Manage secrets without disclosure

**User story:** As a secret administrator, I want to create, update, and delete
named secrets and keys, so that workflows can use sensitive values without
revealing them to viewers.

**Acceptance criteria:**

- **AC-INT-004.1:** When an authorized administrator stores a secret value,
  SuperPlane shall confirm the key exists without returning the stored clear
  value in later listings.
- **AC-INT-004.2:** When a secret key is deleted, subsequent workflow use of
  that key shall fail safely without revealing its former value.

## Traceability

- **Product context:** [supported integration model](../../README.md#supported-integrations)
- **API evidence:** [organization integration lifecycle](../../protos/organizations.proto),
  [integration catalog](../../protos/integrations.proto),
  [component catalog](../../protos/components.proto), and
  [secrets service](../../protos/secrets.proto)
- **UI evidence:** [secret query hook](../../web_src/src/hooks/useSecrets.ts)
- **Behavior evidence:** [secret management](../../test/e2e/secrets_test.go)
- **Feature blueprints:**
  [Integrations and Event Ingestion](../blueprints/features/integrations-and-event-ingestion.feature.md)
  and
  [Secrets and Runtime Configuration](../blueprints/features/secrets-and-runtime-configuration.feature.md)

## Open Questions

- Which users may inspect integration configuration metadata versus rotate its
  credentials?
- What secret rotation and versioning guarantees are required for in-flight
  runs?
