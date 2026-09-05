# Selectable LLM models

> Status: Draft  
> Audience: Product and engineering

This playbook is the source for SuperPlane model picklists. One list serves
Create with an Agent, SuperPlane node fields, and later pickers.

## Locked decisions

1. **One list function and one RPC.** Every picklist loads
   `ListSelectableLLMModels`. Callers filter. They do not invent a second
   catalog.
2. **Item shape is source, provider, model, key, and label.** The key is
   `source::provider::model`. The label is the technical name
   `provider/model`. OpenRouter labels stay as the model id.
3. **Source is SuperPlane-hosted versus the user keys.** Source is not
   "Integration". SuperPlane-hosted rows use source name SuperPlane. User-key
   rows use source name Your keys.
4. **Duplicates stay when source differs.** The same model id can appear twice
   when one row is hosted and one row is BYOK.
5. **Create with an Agent is the first picker.** The title bar shows
   `Using {label}`. A pick reloads the agent, keeps the session, and records
   spend per execution and model.
6. **Later pickers reuse the same list.** SuperPlane node fields, draft start,
   and BYOK runner fields filter this list. They do not build a new catalog.

## Goal

A user picks a model from one canonical list. SuperPlane-hosted models and
selected BYOK models appear together. The picker shows who pays for the run.

Create with an Agent is the first product surface that loads this list, shows
it, and executes the pick.

## What exists today

Do not reinvent these pieces:

- Installation hosted allowlists already gate SuperPlane-hosted models.
- Organization BYOK selected lists already store models the user chose.
- A factory can further subset those lists.
- `ResolveSelectableLLMModels` already applies those allowlists per provider
  and funding source.
- Factory spend is `workspace_usage_events`. Create with an Agent spend is
  factory usage, not canvas Agent Tokens.
- SuperPlane execute already stamps `hostedProvider`, `model`, and
  `credentials.source=hosted` on the run configuration.

Do not include live BYOK candidate catalogs in this list. Empty BYOK is
normal until the organization selected list exists.

## Product rules

| Topic | Rule |
| --- | --- |
| Catalog | One RPC. Callers pass `factory_id` and filter `sources`. |
| Key | `hosted::anthropic::claude-sonnet-4-6` or `byok::openai::gpt-5`. |
| Label | `anthropic/claude-sonnet-4-6`. OpenRouter uses the model id only. |
| Source names | SuperPlane for hosted. Your keys for BYOK. |
| Duplicates | Keep both rows when source differs. |
| Factory subset | Optional. When set, hide models the factory does not allow. |
| Create with an Agent | Title-bar `Using {label}` with a wavy underline. |
| Reload | Keep the planning session. Start a new machine. Keep the transcript. |
| Shared canvas | Do not write the pick onto the live `planning-agent` node. |
| Spend | Old execution keeps old spend. New execution records new spend. |

## Domain model

```
Installation hosted allowlists
Organization BYOK selected lists
Optional factory subset
        └── ListSelectableLLMModels
              └── picker (Create with an Agent, SuperPlane node, later fields)
```

Each item:

- `source`: `{ id: hosted \| byok, name: SuperPlane \| Your keys }`
- `provider`: `{ id, name }` for Anthropic, OpenAI, or OpenRouter
- `model`: `{ id, name }`
- `key`: `source::provider::model`
- `label`: `provider/model` (OpenRouter: model id)

Sort by label, then source id, then key.

## Create with an Agent

The picker lives in the dialog title bar, left of machine status and End
session.

Default shown value, in this order:

1. Session stored key.
2. Current planning-agent node model.
3. Instance SuperPlane agent model.

A pick does not call End session. SuperPlane:

- Persists `selectable_model_key` on the planning session.
- Cancels the current canvas run.
- Starts a new canvas run on the same session.
- Points that run at a session-scoped unpublished canvas version.
- Leaves the live planning canvas unchanged.
- Seeds the new prompt with a short rewind of prior session messages.

Map the key to a runner:

- `hosted` → `runnerSuperPlane`
- `byok` + `anthropic` → `runnerClaudeCode`
- `byok` + `openai` → `runnerCodex`
- `byok` + `openrouter` → `runnerOpenRouter`

The cancelled run must reach a terminal broker task and persist usage before
the new run starts. Failed or cancelled tokens already consumed stay on the
ledger.

## SuperPlane node

The SuperPlane Model field is a second consumer. It loads the same list with
`sources: [hosted]` and stores the three-part key. Execute parses that key
and runs the SuperPlane-hosted model.
