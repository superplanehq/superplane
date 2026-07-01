# Integration Setup Flow

This document captures the design behind the new integration setup flow. The core contract lives in `pkg/core/integration_setup_provider.go`, with the current runtime wired through organization integration actions and integration-specific setup providers.

## Goals

- Let each integration own its setup journey instead of forcing all integrations through one static configuration form.
- Support multi-step setup flows with instructions, input fields, redirects, completion screens, and back navigation.
- Separate sensitive secrets from user-visible properties.
- Allow users to request only the capabilities they need, and allow integrations to enable those capabilities only after the required setup work is complete.
- Re-enter setup when an editable property, editable secret, or newly requested capability requires additional work.

## Core Model

The setup flow is implemented by `core.IntegrationSetupProvider`.

Each provider declares all capabilities available for that integration through `CapabilityGroups()`. A capability is an action or trigger with its label, description, configuration fields, and action output channels. Capability groups are presentation groups for the setup UI and CLI; the persisted state is still tracked per capability.

Each provider also owns the setup state machine:

- `FirstStep(ctx)` returns the initial pending step.
- `OnStepSubmit(ctx)` handles submitted inputs for the current step and returns the next step.
- `OnStepRevert(ctx)` rolls back side effects from the last successfully submitted step.
- `OnPropertyUpdate(ctx)` handles updates to editable user-visible properties.
- `OnSecretUpdate(ctx)` handles updates to editable sensitive values.
- `OnCapabilityUpdate(ctx)` handles requests for new capabilities after the integration already exists.

Handlers receive storage and service contexts instead of loading state directly:

- `Properties` stores non-sensitive integration data visible to users.
- `Secrets` stores sensitive data, encrypted through the integration secret storage.
- `Capabilities` reads requested capabilities and moves capabilities between states.
- `HTTP`, `Logger`, `BaseURL`, and `WebhooksBaseURL` provide runtime services needed by setup flows.

## Setup Steps

A setup step is a serializable UI/CLI instruction for what the user should do next.

Step types are:

- `inputs`: display markdown instructions and collect typed `configuration.Field` inputs.
- `redirectPrompt`: ask the user to continue through an external URL, optionally with form data.
- `done`: display completion instructions. Submitting a `done` step clears setup state and marks the integration ready.

Every step has a stable machine-readable `Name`, a human-readable `Label`, optional markdown `Instructions`, and type-specific data.

## Persisted State

New-flow integrations store setup-specific data on `models.Integration`:

- `SetupState`: the current pending step plus previous steps.
- `Properties`: user-visible non-sensitive setup results.
- `Capabilities`: one state per declared capability.

Secrets are stored separately in `app_installation_secrets` as `models.IntegrationSecret`, so sensitive values are not mixed into regular integration configuration.

The old integration fields, such as `Configuration`, `Metadata`, and `BrowserAction`, still exist for legacy setup. They should become less important as integrations move to `IntegrationSetupProvider`.

## Capability States

Capabilities are tracked with four states:

- `requested`: the user asked for the capability, but setup has not exposed it yet.
- `enabled`: the capability is available for use.
- `disabled`: the capability was available but the user manually disabled it.
- `unavailable`: the integration supports the capability, but it was not requested for this installation.

When a new integration is created through the new flow, requested capability names are validated against the provider's declared capabilities. Requested capabilities start as `requested`; all others start as `unavailable`. The setup provider decides when requested capabilities can be enabled.

For example, Semaphore enables requested capabilities after a valid API token is stored and verified. GitHub can ask the user to update token permissions before newly requested capabilities become available.

## Editing After Setup

Properties and secrets can be marked editable by the provider. When users update editable values, the corresponding provider callback receives the new value.

The provider can either:

- validate and persist the new value without returning a setup step, or
- return a setup step to restart a focused setup flow.

For example, a secret update can validate a replacement API token before storing it. A property update could require the user to reconnect, select a resource again, or reauthorize externally.

## Updating Capabilities After Setup

Users can update capability states after the integration exists.

Simple enable and disable changes update stored capability states directly. Newly requested capabilities are delegated to `OnCapabilityUpdate`, because the provider may need to validate permissions, collect new credentials, create webhooks, or show instructions before enabling them.

If `OnCapabilityUpdate` returns no step, the handler persists whatever capability states the provider set. If it returns a step, SuperPlane stores a new setup state so the integration can re-enter setup for that capability expansion.

## Provider Responsibilities

An integration setup provider should:

- Declare all actions and triggers in `CapabilityGroups()`.
- Keep step names stable because they are persisted and used for dispatch.
- Store user-visible non-sensitive values as properties.
- Store credentials, tokens, private keys, and webhook secrets as secrets.
- Mark properties and secrets editable only when updates are supported.
- Validate external credentials before enabling capabilities.
- Enable requested capabilities only after setup has completed the required external work.
- Clean up step side effects in `OnStepRevert`.
- Return `done` when the user should see a completion screen, or `nil` when setup can finish without one.
