# Inline configuration assistant (AI field suggest)

## Overview

This PRD describes **in-config assistance**: a **stateless**, per-field flow in the workflow node
sidebar where users describe what they want in natural language and receive a **suggested
value** for a single configuration field (confirm or discard). Suggestions are produced by a
**dedicated** LLM path (PydanticAI + Anthropic) behind SuperPlane’s public API.

This is **not** AI Builder chat, session history, or canvas-mutating tools.  
It does **not** persist conversation state or replace expression autocomplete / local preview.

The assistant is a **single LLM call** with **no tools**: context is the **system prompt**,
**instruction**, and **`field_context_json`** (field metadata, current value, expression payload
shape, etc.). `canvas_id` / `node_id` are for auth and routing, not agent-side data fetches.

## Problem Statement

Power users editing node configuration—especially **expressions** and long text—must know
expr-lang conventions, payload shape, and component semantics. Trial-and-error is slow even
with autocomplete and preview.

We want a **narrow** assistive surface: “describe intent → see proposed string → apply to
this field” without opening the full builder or a separate chat product.

## Goals

1. Offer **one-shot suggest** for eligible fields on the workflow **Settings** / configuration
   UI, gated by product and permissions.
2. Keep **trust boundaries clear**: browser → authenticated SuperPlane API → agent HTTP with
   **scoped JWT** (`purpose: config-assistant`), separate from AI Builder tokens.
3. Run a **small** Python agent (`agent/src/config_assistant/`) with **structured output**
   (`value`, optional `explanation`); **no** import of the builder’s `agent.py`, **no** shared
   agent tool loop, **no** tools—context only via **`field_context_json`** (+ instruction and
   system prompt).
4. Centralize client wiring in **`lib/configAssistantSuggest.ts`** and
   **`useConfigAssistantSuggest`** so request shape and errors are not scattered across UI.

## Non-Goals

- Streaming tokens in the assistant panel (unary JSON).
- Shared chat transcript with AI Builder or reuse of builder streaming routes.
- **Multi-turn** dialogue inside the assistant panel (each **Generate** is independent).
- Catalog-driven per-field flags on shared `Field` schema in the near term (see **Decisions**:
   **frontend type allowlist** only for now).
- Server-side redaction of every secret in `field_context_json` beyond size limits (hardening
  may follow).
- **Structured / non-string** primary `value` in the API (e.g. JSON object as the core contract);
  responses stay **string-only**; UI coercion stays client-side where needed.

## Decisions

1. **Org-level “AI off”:** Align with AI Builder—when the organization disables AI features,
   field suggest MUST fail with **403** (or equivalent), not silently call the model.
2. **Field eligibility:** Keep the **frontend allowlist** (`isConfigAssistantSupportedField` in
   `web_src/src/lib/configAssistantFields.ts`); defer catalog-level `Field` schema opt-in until
   the product stabilizes.
3. **Multi-turn / streaming:** **No** multi-turn in the panel; **no** streaming in v1—single
   instruction in, single structured response out.
4. **Response shape:** **`value` remains a string** end-to-end in the public API and agent
   contract; no parallel structured-value field required for v1.

## Scope and Semantics

### Eligible fields (frontend allowlist)

Assistance is shown only when the feature is enabled **and** the field passes
`isConfigAssistantSupportedField` (`web_src/src/lib/configAssistantFields.ts`):

- **Allowed types:** `expression`, `string`, `text`, `url`, `cron`.
- **`integration-resource`:** single-value (non-multi) fields only; assistant appears in
  **Expression** tab mode, not Fixed picker mode.
- **Never:** `sensitive: true`; **not** `select`, `multi-select`, `number`, or `boolean` (fixed
  catalogs or poor NL UX).

### UX contract (`InlineFieldAssistant`)

- Trigger (sparkle) beside the field label row where applicable.
- Panel: instruction textarea → **Generate** → readonly suggested value + optional explanation →
  **Use this value** / cancel.
- Applying a value runs normal field `onChange`; save-time validation remains the backstop.

### Request payload (conceptual)

Public API accepts `canvas_id`, `node_id`, `instruction`, and `field_context_json` (JSON string).

**`field_context_json`** MUST carry everything the model needs to propose a good **string**
value without tools, including at minimum:

- Structured **field** metadata (from `buildConfigAssistantFieldContext`: name, label, type,
  description, placeholder, required, `typeOptions` when present).
- **currentValue** (or equivalent) for the field being edited.
- For **expression** (and related) fields, **autocompleteExample** (or equivalent)—the same shape
  the UI uses for Monaco autocomplete: named upstream node payloads, `__root`,
  `__previousByDepth`, `__nodeNames`, etc., derived from the workflow graph client-side—not fetched
  by the agent at runtime.

Go enforces caps: instruction length (runes) and max `field_context_json` bytes.

### Response

- `value` (string): proposed field value; must be non-empty for success.
- `explanation` (optional): short human-readable note.

## Public API (Go)

- **HTTP:** `POST /api/v1/agents/config/suggest-field` (gRPC-Gateway for
  `Agents.SuggestConfigurationField` on the unified agents service).
- **Auth:** Same session authentication as the rest of the app.
- **AuthZ:** User must be allowed to **update** the canvas (same bar as editing the workflow);
  plus **org AI enabled** aligned with AI Builder (**403** when AI is disabled for the org).
- **Agent call:** Mint short-lived JWT with **`purpose: config-assistant`** (distinct from
  builder’s `agent-builder`); forward JSON to the agent HTTP service (path may remain e.g.
  **`{AGENT_HTTP_URL}/config-assistant/suggest`**—internal to SuperPlane, not the public API
  path).
- **Scopes:** Include `canvases:read:<canvas_id>` (and related org/integration read checks as
  implemented) so the agent can validate the JWT without builder-only semantics.

Proto and generated code live under `protos/agents.proto` and `pkg/protos/agents/`; the
**documented public URL** is **`/api/v1/agents/config/suggest-field`**.

## Agent service (Python)

- **Mount:** Same FastAPI app as the builder; router mounted from `repl_web` at an internal path
  (e.g. **`POST /config-assistant/suggest`**).
- **Validate:** JWT (shared validator in `agent/src/ai/jwt.py` with purpose allowlist:
  `agent-builder`, `config-assistant`).
- **Run:** PydanticAI agent in `agent/src/config_assistant/`; model from
  **`CONFIG_ASSISTANT_AI_MODEL`** or **`AI_MODEL`**; **`ANTHROPIC_API_KEY`** for Anthropic.
- **No** builder tools, **no** tool calling for this path: single LLM pass over instruction +
  parsed **`field_context_json`** (+ system prompt). **No** `agent.py` dependency for this path in
  v1.

## Authorization and configuration

- **SuperPlane:** Canvas update permission on the target canvas; org-level AI policy consistent
  with AI Builder; `AGENT_HTTP_URL` must resolve to the agent HTTP service (in Docker Compose,
  use the agent service hostname, not `localhost` from inside the app container).
- **Agent:** `JWT_SECRET` must match the app; optional separate model string for cost/latency
  tuning vs builder.

## Frontend integration

- **Gate:** `VITE_ENABLE_INLINE_CONFIG_ASSISTANT` (and read-only sidebar) for visibility;
  **server** AuthZ is authoritative.
- **Hook:** `useConfigAssistantSuggest` supplies `isFieldAssistantEnabled` and
  `getSuggestFieldValue(field, getCurrentValue)` for `SettingsTab` →
  `ConfigurationFieldRenderer`.
- **Mock path:** When canvas/node/org context is missing, client uses a short delayed mock so
  Storybook / partial contexts can still exercise UI.

## Example flow

1. User opens a node on a workflow canvas with assistant enabled.
2. User focuses an **expression** field, opens sparkle, enters “filter to open PRs only”.
3. UI calls `POST /api/v1/agents/config/suggest-field` with `field_context_json` (including
   autocomplete-style upstream payload) and instruction.
4. Go authorizes (canvas update + org AI allowed), mints `config-assistant` JWT, POSTs to agent.
5. Agent runs **one** model call—no tools—using instruction + context JSON + system prompt;
   returns `{ value, explanation }`; user confirms; value is written to configuration and
   autosave/save rules apply as usual.

## Acceptance Criteria

1. Eligible fields show the assistant; **sensitive** and disallowed types do not.
2. Successful suggest returns a non-empty string **`value`**; errors surface in the panel without
   breaking the rest of the sidebar.
3. Builder JWTs cannot be used for the config-assistant agent route and vice versa (**purpose**
   separation).
4. User without canvas **update** cannot obtain successful suggests for that canvas.
5. When org AI is disabled (same policy as AI Builder), suggest returns **403** (or documented
   equivalent).
6. Agent and app share JWT signing secret; misconfiguration yields clear 503/4xx behavior
   operable from logs.

## Risks and Mitigations

- **Risk:** Context JSON leaks secrets or PII to the model.  
  **Mitigation:** exclude sensitive fields; cap payload size; document trust boundary; add
  redaction in Go in a follow-up.

- **Risk:** Model proposes invalid expression or wrong type.  
  **Mitigation:** user must confirm; existing validation on save; prompt guidance in
  `agent/src/config_assistant/system_prompt.txt`.

- **Risk:** Cost / abuse.  
  **Mitigation:** instruction and context limits; rate limits; org AI disable aligned with
  Builder.

- **Risk:** Operator misconfigures `AGENT_HTTP_URL`.  
  **Mitigation:** document Compose defaults; log failed forwards from Go.

## Related

- Implementation tracks work such as [issue #3714](https://github.com/superplanehq/superplane/issues/3714).
