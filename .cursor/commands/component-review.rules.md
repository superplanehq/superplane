# Component Review Rules

Add new rules to this list in the same format. Keep each rule focused and testable. Categories define ordering and scope.

## Product functionality

Focus on externally visible behavior in the UI, not internal function details.

### Registration

The component is registered in the application. 
For core compoents: with `registry.RegisterComponent`
For integrations: listed in the integration's `Components()` or `Triggers()` output.

### Naming

- For core components: uses the same name as the registration name
- For integration components: name should follow the format `<integration>.<name>`, e.g. `github.getIssue` vs `github.GetIssue`
- A component name should always use camel-case, and should never contain spaces, or underscore
- If the component is a trigger, it should start with `on`
- Trigger names should use the resource names that they reference, not actions, unless the trigger itself is about an specific action. For example, `github.onIssueComment` instead of `github.onIssueCommented`

### Label

- The component label shown in the UI is non-empty and human-readable
- Uses Title Case
- No raw slug casing

### Description

The component description shown in the UI is non-empty, user-facing. Short, clear and concise.

### Documentation

- The component documentation shown in the UI is non-empty.
- It is valid markdown
- If it uses titles, the biggest level should be ##

### Icon

- The component icon shown in the UI is non-empty and maps to a valid UI icon
- Either Lucide Icon slug
- Or an existing custom asset

### Color

The component color shown in the UI is non-empty and consistent with existing component color usage.

### Example output

- The example output shown in the UI embeds an example 
- JSON file (e.g., `example_output.json`).
- Example JSON files are valid and match the emitted payload structure.

### Configuration fields

- Every configuration field shown in the UI has a: `Name`, `Label`, `Type`, and `Description`
- Required fields are marked `Required: true`.
- Required fields should always be placed before optional fields
- When possible, configuration never asks users to enter IDs that aren't easily available to them (e.g., requiring a Discord channel ID).
- When users can choose from existing resources (e.g., a GitHub repository), always use `FieldTypeIntegrationResource`. Example: instead of having a `channelId` field of type `FieldTypeString`, use a `FieldTypeIntegrationResource` configuration field.
- For trigger filters and component configuration that use equality, non-equality, or regex matching, prefer `configuration.FieldTypeAnyPredicateList` and match values via `configuration.MatchesAnyPredicate`. Avoid ad-hoc wildcard or comma parsing unless `any-predicate-list` cannot express the requirement. If `configuration.FieldTypeAnyPredicateList` cannot meet your requirements, look for ways to extend it before developing a specific implementation.
- if filters are part of the trigger, we should always have a default filter for the most common use case. That makes it easier for the user to configure it, and we don't produce unnecessary events to the system. For example, the default `github.onPush` refs filter is for commits on the main branch.

### Output channels

The output channels shown in the UI include at least one channel (or rely on default), and channel names/labels are non-empty.

### Setup

- If the component/trigger uses an `FieldTypeIntegrationResource` configuration field, `Setup()` must verify that the resource being referenced exists, and once verified, information about it must be stored in a struct in the component/trigger metadata.

### Webhooks

- If the webhook is not configured through the integration, use `ctx.Webhook.Setup()`. If the webhook is configured through the integration, `ctx.Integration.RequestWebhook()` and implement the integration's `SetupWebhook`, `CleanupWebhook`
- We should always aim to share webhooks between components, if they use the same underlying event configuration. Use `CompareWebhookConfig` for that. For example, if we have two `github.onPush` triggers, one for main branch, and one for release branches, both of those triggers use the same webhook in GitHub.

### Triggers

- A trigger is always scoped to a (1) specific resource type, (2) specific resource, (3) some additional things. Examples:
  - `semaphore.onPipelineDone`: we select the specific project we want to listen to
  - `github.onPush`: we select the repository we want to listen to
  - `pagerduty.onIncident`: we select the service

### Security

- Components should always execute HTTP requests using the `HTTPContext` available to them, and never use `net/http` to do so
- Components should never import `pkg/models` and interact with database directly, only through methods provided through core interfaces
- HandleWebhook() implementations in components/triggers should always verify that the requests are authenticated using the secret in the webhook
- HandleRequest() implementations in integrations should always verify that the requests are authenticated using the secret in the webhook

## Code Quality

### Unit testing

- Static methods like `Configuration()`, `Label()`, `Name()` do not need to be unit tested.
- Do not make dummy implementations of the `pkg/core` interfaces in unit tests. Use contexts already available in [test/support/contexts](https://github.com/superplanehq/superplane/blob/main/test/support/contexts/contexts.go) for that.
- Tests cover validation failures and error handling paths.
- For `Component` interface implementations, tests for `Setup()` and `Execute()` must be written. If the component has `Actions()`, they must be unit tested as well
- For `Trigger` interface implementations, tests for `Setup()` and `HandleWebhook()` must be written. If the component has `Actions()`, they must be unit tested as well
- For `Integration` interface implementations, tests for `Sync` must be written. If the component has `Actions()`, they must be unit tested as well

### General principles

- Favor early returns and the use of helper functions over nested `if/else` blocks.
- Reusable strings (names, payload types) are defined as constants when repeated.
- New imports are used; no dead code or unused helpers.

### Golang

- Prefer `any` over `interface{}` types
- When checking for the existence of an item on a list, use `slice.Contains` or `slice.ContainsFunc`
- When naming variables, avoid names like `*Str` or `*UUID`; Go is a typed language, we don't need types in the variables names
- When writing tests that require specific timestamps to be used, always use timestamps based off of `time.Now()`, instead of absolute times created with `time.Date`
- **Check transaction usage**: Any DB access in transactional flows uses `*InTransaction()` variants, never `database.Conn()`

### TypeScript

- **No implicit any**: use explicit types for inline handler parameters
- **No `any` abuse**: avoid `as any` / `@ts-ignore` unless justified (prefer narrow types or `@ts-expect-error` with comment).

## Copy

- **User-facing text**: all user-facing strings are short, clear, and consise
- **Clarity**: Labels, descriptions, and docs avoid internal jargon and explain intent.
- **Consistency**: Terms used in docs match configuration labels and output fields.
- **Formatting**: Markdown in the component documentation renders cleanly (no malformed code fences or headings).
