---
name: superplane-new-intake
description: >-
  Use when adding a new factory intake source, such as Linear, Notion, or
  Datadog. Ask which source, then collect the product decisions and API facts
  required before implementation. Covers trigger, binding, seed, search,
  settings, and the Backlog Add intake flow.
---

# New factory intake

Use this skill before you add a factory intake source. An intake listens to an
external source and creates a Backlog task. The canvas graph owns the
behavior. The intake row owns the identity.

Do not write intake code until the human confirms the spec at the end of this
skill. If they already named the source, start at **Look up the source**. If
they did not, ask one question and stop:

> Which intake do you want to add? For example, Linear.

## Look up the source

Search the repo before you ask the human anything the code already answers.

- Integration: `pkg/integrations/<name>/` and `docs/components/<Name>.mdx`.
- Trigger: a component such as `linear.onIssue`. Read its configuration fields
  and an example payload (`example_data_*.json` or the trigger docs).
- Intake: `FactoryIntakeSource*` in `pkg/models/factory_intake.go` and
  `ADD_INTAKE_TEMPLATES` in
  `web_src/src/pages/factories/pages/addIntakeTemplates.ts`.

Tell the human what already exists, in two or three sentences. Then walk the
checklist. Skip a question when the repo already answers it, and say the fact
you will use.

PagerDuty is not a finished intake. It has a source enum and a canvas
template, and it is not in the Add intake picker. Do not copy it as the
pattern for a new source.

Closest finished intakes:

- The integration and trigger already exist: Dependabot alerts (`#7833`).
- The source needs its own connection and setup wizard: Jira issues
  (`#7534`), Sentry exceptions (`#7484`), Productive tasks (`#7030`).

Read that closest intake before you edit. Match its files. Do not invent a
second way to register a source.

## How to ask

- One checklist group per turn. Wait for the answer.
- Ask only what this source still needs. Do not repeat the whole list.
- Propose a default when past intakes already chose one. Ask the human to
  accept it or change it.
- If the human already gave a full spec, map it onto the checklist, list the
  gaps, and ask only for the gaps.
- User-facing names and helper text follow
  [ui-copy](../../.agents/skills/ui-copy/SKILL.md) and
  [simplified-technical-english](../../.agents/skills/simplified-technical-english/SKILL.md).
  The product name is SuperPlane.

Required before any code:

- Events that create a task
- One task per what (one issue, one package, one alert)
- Title fields, body fields, and the browse URL field
- Resource the trigger listens on, and the configuration key name
- Filters and their defaults
- First import: on or off, item cap, and whether the user can skip it
- Feature flag: on or off for the Add intake entry
- Write-back when the task completes: yes or no

You may propose, and the human confirms:

- Source id, kebab-case, such as `linear-issues`
- Picker name and one-sentence description
- Pause, resume, and delete (default: yes, same as current intakes)
- Icon (reuse the integration icon when one exists)

## Checklist

### 1. What becomes a task

- Which events create a task? Examples: created, reopened, assigned, a
  label added. Name the trigger action values, not only the product words.
- Does a later event update an open task, or create another task?
- Which items never become a task? Examples: closed, key tasks, low severity.

### 2. Task text and origin

- Title. Name the payload fields. Example: Jira uses
  `issue.key` and `issue.fields.summary`.
- Body. Name the payload fields. Use the plain-text field when the API stores
  rich text.
- Browse URL field for the task origin. Past intakes store this URL and skip
  a later import of the same URL.
- Run title path for the intake run list. Each source nests the title in a
  different place. See `intakeRunTitle` in
  `pkg/grpc/actions/factories/list_factory_intake_runs.go`.

### 3. Setup choices

- Which integration installation does the trigger use?
- Which resource does it listen on? GitHub uses `repository`. Jira, Sentry,
  and Productive use `project`. Linear uses `team` on `linear.onIssue`.
  Record the configuration key. `TriggerResourceID` in
  `pkg/grpc/actions/factories/intake_graph.go` reads `repository` and
  `project` only. A new key must be added there or settings lose the
  resource.
- Which filters appear in the setup wizard and in intake settings? Examples:
  labels, assignment, severity, level, task list. Empty selection means every
  value, unless the human says otherwise.
- Default for each filter.
- Can the user reconnect a missing integration from settings? Jira, Sentry,
  and Productive can. GitHub uses the workspace repository from onboarding.

### 4. First import

- Import existing items when the intake is created, or wait for the next
  event?
- Cap. Current caps: GitHub issues 30, Jira 10, Sentry 10, Productive 10.
  Dependabot does not auto-import. The user picks packages, and create sets
  `skipInitialImport`.
- Offer **Skip import** in the wizard? Jira, Sentry, and Productive do.
- Search and import one item by id, so the wizard and **Import** can read the
  source. The list API and the trigger payload must be the same shape.
- A setup list that runs before the intake exists needs its own RPC, as
  Dependabot does in `search_dependabot_intake_setup_items.go`. Prefer
  `SearchFactoryIntakeItems` after the intake exists.

### 5. Connection

Skip this group when the integration and a suitable trigger already exist.
Say which trigger you will reuse. Do not add a second trigger for the same
event.

Ask this group only when the integration or the trigger is missing:

- Auth: OAuth app, API token, or an existing app installation.
- Delivery: webhook or poll. Current live intakes use webhooks.
- Who can connect. Linear webhooks need a workspace admin token.
- List API for search and seed, and get-by-id API for import.
- Example webhook payload, including the payload type string the trigger
  emits. Seed events must use that same type.
- Token scopes. Read is enough for intake. Write is required only when the
  human asked for write-back.
- Public app env vars, when the product installs a SuperPlane-owned app
  (Sentry, Jira). Name the variables. Document them in
  `docs/contributing/connecting-to-3rdparty-services-from-development.md`
  and `.env.example`.

### 6. After the task exists

Default is no write-back and no attachment copy. Ask anyway. Do not add
either unless the human says yes.

- When the task completes, does SuperPlane change the source item? Jira can
  move the issue to a status column. That landed after the first intake
  commit. It is not part of the first slice unless requested.
- Copy attachments or images onto the task? Jira and Productive did this in
  later commits. Skip it on the first slice unless requested.
- Webhook health. Report a webhook that is pending or failed when the source
  can look connected and still receive nothing. Jira and Productive do this
  in `intake_graph.go`.

### 7. Release

- Ship the Add intake entry behind an organization feature flag. That is the
  pattern for Sentry, Jira, Productive, and Dependabot. GitHub issues has no
  flag. Default for a new source: flag on, entry shows **Coming soon** until
  the flag is enabled.
- Picker name, description, and icon.
- Setup page title, helper text, skip-import label, and create-error text.

## Confirm, then implement

Repeat the answers as a short spec. Include the trigger component, source id,
events, title, body, origin URL, resource key, filters, seed cap, feature
flag, and write-back. Ask the human to confirm. Start code only after they
confirm.

Generated files stay gitignored. Edit `protos/factories.proto`, then run
`make pb.gen`. Do not hand-edit `pkg/protos/`, `web_src/src/api-client/`, or
`api/`. Append proto enum values and settings fields so numbers stay
contiguous. Run `make check.proto.field.numbers`. A new source string does
not need a migration. The source column is already text.

The generated graph is: trigger, optional filter, create task. Do not add an
analysis node. Scoring happens on the Backlog canvas.

### Register the source

- `pkg/models/factory_intake.go`: constant, `factoryIntakeSources`, and
  `pkg/models/factory_intake_test.go`.
- `protos/factories.proto`: `FactoryIntake.Source` and, when filters are new,
  `FactoryIntake.Settings` fields.
- `pkg/grpc/actions/factories/serialization.go`: `serializeFactoryIntakeSource`
  and `parseFactoryIntakeSource`.
- `pkg/grpc/actions/factories/factory_velocity_report.go`: label and series
  order, when this source should appear on the velocity report. Jira and
  Productive are absent there today. Ask before you add a band.

### Build the canvas

- `pkg/grpc/actions/factories/intake_template.go`: `intakeSpecsBySource`
  (trigger component, trigger name, default configuration, create title,
  create description).
- `pkg/grpc/actions/factories/create_factory_intake.go`: default settings
  switch.
- `pkg/grpc/actions/factories/intake_settings.go`: settings fields, defaults,
  filter expression, and proto parse/serialize. Filters live in the graph.
  Do not store them a second time on the intake row.
- `pkg/grpc/actions/factories/intake_binding.go`: integration app name and
  resource configuration. An unbound trigger registers no webhook.
- `pkg/grpc/actions/factories/intake_graph.go`: resource key and, when
  needed, webhook health and rebind.

### Read live items

- `pkg/grpc/actions/factories/intake_seed.go`: list existing items, emit the
  trigger payload type, skip items that already have a task, honor the cap.
  Record skipped, failed, and completed on the intake.
- `pkg/grpc/actions/factories/intake_items.go`: register the trigger
  component for Search, Get, and origin-id matching.
- `pkg/grpc/actions/factories/list_factory_intake_runs.go`: `intakeRunTitle`.
- `pkg/grpc/actions/run_title_defaults.go`: canvas run title for the trigger,
  when the trigger has no entry yet.
- Merge into one open task only when the human asked for that. Dependabot
  does it in `pkg/workers/contexts/factory_context.go`. Other sources create
  one task per item.

### Trigger

Reuse the existing trigger when one matches the events. When it does not,
add the component under `pkg/integrations/<name>/`, register it, add an
example payload, and update `docs/components/<Name>.mdx`. Follow
`docs/contributing/component-implementations.md`.

### Feature flag and UI

- `pkg/features/features.go` and `pkg/features/features_test.go`.
- `web_src/src/lib/experimentalFeatures.ts`.
- `web_src/src/pages/factories/pages/addIntakeTemplates.ts`: picker entry and
  `featureId`.
- `web_src/src/pages/factories/pages/lineIntakeModel.ts`: source id, API enum
  map, listen and evaluate copy.
- `web_src/src/pages/factories/pages/lineIntakeCanvas.ts`.
- `web_src/src/pages/factories/pages/intakeSourceSettingsModel.ts`: settings,
  pause, and delete.
- `web_src/src/pages/factories/pages/IntakeSourceSettingsPopup.tsx` and a
  filter-fields component for this source.
- Setup wizard, page, copy, and route. Wire the picker in `LinesPage.tsx`.
  Add the path in `web_src/src/pages/factories/lib/factoryPagePaths.ts` and
  the route in `web_src/src/App.tsx`. Export the page from
  `web_src/src/pages/factories/index.ts`.
- `web_src/src/pages/factories/pages/work-order-split-run/splitRunSource.ts`
  when task cards should show this source.
- Icon in `web_src/src/assets/icons/integrations/` when the integration has
  none.

Match the newest setup wizard (Dependabot or Jira). Use shadcn form
components. Put copy in a `*IntakeSetupCopy.ts` file next to the wizard.

### Tests

Add the source to the tests that already switch on source:

- `pkg/models/factory_intake_test.go`
- `pkg/grpc/actions/factories/intake_template_test.go`
- `pkg/grpc/actions/factories/intake_settings_test.go`
- `pkg/grpc/actions/factories/intake_seed_test.go` when the source seeds
- `pkg/grpc/actions/factories/intake_items_test.go`
- `pkg/grpc/actions/factories/list_factory_intake_runs_test.go`
- UI specs for the picker, setup dialog, filter fields, and settings

## Linear, when they name it

Re-read `pkg/integrations/linear/on_issue.go` before you quote field paths.

The repo already has the Linear integration and `linear.onIssue`. There is
no `linear-issues` intake. Reuse that trigger. Do not ask the human to
describe Linear auth.

Facts you can state, then still confirm the product choices:

- Resource key: `team`
- Actions: `create`, `update`, `remove`. Default on the trigger is `create`.
- Payload type: `linear.issue`. The trigger emits the Linear webhook body.
- Title: `{{ root().data.data.title }}`
- Identifier: `{{ root().data.data.identifier }}` (for example `ENG-142`)
- Body: `{{ root().data.data.description }}`
- Browse URL: `{{ root().data.url }}`
- Webhooks require a workspace admin token

Still ask: which actions create a task, title format, filters (labels, state,
priority), seed cap, skip import, feature flag, and write-back.
