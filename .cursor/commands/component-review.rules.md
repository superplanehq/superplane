# Component Review Rules

Add new rules to this list in the same format. Keep each rule focused and testable. Categories define ordering and scope.

## Product functionality

Focus on externally visible behavior in the UI, not internal function details.

### Registration

The component is registered in the application. 
For core compoents: with `registry.RegisterComponent`
For integrations: listed in the integration's `Components()` or `Triggers()` output.

### Naming

The component name shown in the UI is non-empty and matches:
    - Core Components: uses the same name as the registration name
    - For integration: 
        - Starts with <integration>
        - Followed by a dot
        - Followed by the name od the component
        - First letter of the component should be lowercase, e.g. github.getIssue vs github.GetIssue
        - The name should be camel-case
        - Should not contain spaces, or underscores
        - If the component is a trigger, it should start with `on`

### Label

The component label shown in the UI is non-empty and human-readable 
    - Uses Title Case
    - No raw slug casing

### Description

The component description shown in the UI is non-empty, user-facing. Short, clear and concise.

### Documentation

The component documentation shown in the UI is non-empty.
    - It is valid markdown
    - If it uses titles, the biggest level should be ##

### Icon

The component icon shown in the UI is non-empty and maps to a valid UI icon
    - Either Lucide Icon slug
    - Or an existing custom asset

### Color

The component color shown in the UI is non-empty and consistent with existing component color usage.

### Example output

The example output shown in the UI embeds an example 
    JSON file (e.g., `example_output.json`).

### Configuration fields

Every configuration field shown in the UI has a:
    - `Name`
    - `Label`
    - `Type`
    - `Description`
    - Required fields are marked `Required: true`.

### Configuration inputs

When possible, configuration never asks users to enter IDs that aren't easily
available to them (e.g., requiring a Discord channel ID). 

When users can choose from existing resources (e.g., a GitHub repository), 
prefer a dropdown selector over manual entry.

### Predicate filters

For trigger filters and component configuration that use equality, non-equality, or regex matching, prefer `configuration.FieldTypeAnyPredicateList` and match values via `configuration.MatchesAnyPredicate`. Avoid ad-hoc wildcard or comma parsing unless `any-predicate-list` cannot express the requirement (document the exception in the component/trigger file).

### Output channels

The output channels shown in the UI include at least one channel 
(or rely on default), and channel names/labels are non-empty.

### Setup validation

Setup validation in the UI enforces required configuration and 
shows clear errors for missing inputs.

### Actions/webhooks

If actions or webhooks are available in the UI, 
they validate inputs and show meaningful errors.

## Code quality

### Early returns

Logic favors early returns over nested `else` blocks where applicable.

### Transaction safety

Any DB access in transactional flows uses 
`*InTransaction()` variants, never `database.Conn()`.

### No implicit any

TypeScript (if applicable) uses explicit types for 
inline handler parameters.

### No `any` abuse

Avoid `as any` / `@ts-ignore` unless justified 
(prefer narrow types or `@ts-expect-error` with comment).

### Dependencies

New imports are used; no dead code or unused helpers.

### Constants

Reusable strings (names, payload types) are defined as constants when repeated.

### JSON examples

Example JSON files are valid and match the emitted payload structure.

## Testing

### Unit tests

Component logic has focused tests in `*_test.go`

### Error paths

Tests cover validation failures and error handling paths.

### Setup/execute tests

Tests cover `Setup()` and `Execute()` behavior where applicable.

## Copy

### User-facing text

All user-facing strings are short, clear, and consise

### Clarity

Labels, descriptions, and docs avoid internal jargon and explain intent.

### Consistency

Terms used in docs match configuration labels and output fields.

### Formatting

Markdown in the component documentation renders cleanly (no malformed code fences or headings).
