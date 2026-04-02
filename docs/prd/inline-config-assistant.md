# Inline configuration assistant (AI field suggest)

## Overview

This PRD describes **in-config assistance**: a **stateless**, per-field flow in the workflow node
sidebar where users describe what they want in natural language and receive a **suggested
value** for a single configuration field (confirm or discard). Suggestions are produced by a
**dedicated** LLM path (PydanticAI + Anthropic) behind SuperPlane’s public API.

This is **not** AI Builder chat, session history, or canvas-mutating tools.  
It does **not** persist conversation state or replace expression autocomplete / local preview.

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
   (`value`, optional `explanation`); **no** import of the builder’s `agent.py` or canvas tools.
4. Centralize client wiring in **`lib/configAssistantSuggest.ts`** and
   **`useConfigAssistantSuggest`** so request shape and errors are not scattered across UI.

## Non-Goals

- Streaming tokens in the assistant panel (unary JSON).
- Shared chat transcript with AI Builder or reuse of builder streaming routes.
- Catalog-driven per-field flags on shared `Field` schema (uses a **frontend type allowlist**
   only).
- Server-side redaction of every secret in `field_context_json` beyond size limits.
- Org-level “AI disabled” parity with builder beyond current canvas-edit checks (may be added
   later).

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
Context includes structured **field** metadata (from `buildConfigAssistantFieldContext`),
**currentValue**, and **autocompleteExample** (workflow payload example for expressions).

Go enforces caps: instruction length (runes) and max `field_context_json` bytes.

### Response

- `value` (string): proposed field value; must be non-empty for success.
- `explanation` (optional): short human-readable note.

## Public API (Go)

- **HTTP:** `POST /api/v1/config-assistant/suggest` (gRPC-Gateway for
  `ConfigAssistant.SuggestConfigurationField`).
- **Auth:** Same session authentication as the rest of the app.
- **AuthZ:** User must be allowed to **update** the canvas (same bar as editing the workflow).
- **Agent call:** Mint short-lived JWT with **`purpose: config-assistant`** (distinct from
  builder’s `agent-builder`); forward JSON to **`{AGENT_HTTP_URL}/config-assistant/suggest`**
  with `Authorization: Bearer <JWT>`.
- **Scopes:** Include `canvases:read:<canvas_id>` (and related org/integration read checks as
  implemented) so the agent can validate canvas without reusing builder-only semantics.

Proto and generated code live under `protos/config_assistant.proto` and
`pkg/protos/config_assistant/`.

## Agent service (Python)

- **Mount:** Same FastAPI app as the builder; router mounted from `repl_web` at
  **`POST /config-assistant/suggest`**.
- **Validate:** JWT (shared validator in `agent/src/ai/jwt.py` with purpose allowlist:
  `agent-builder`, `config-assistant`).
- **Run:** PydanticAI agent in `agent/src/config_assistant/`; model from
  **`CONFIG_ASSISTANT_AI_MODEL`** or **`AI_MODEL`**; **`ANTHROPIC_API_KEY`** for Anthropic.
- **No** builder tools, session store, or `agent.py` dependency for this path.

## Authorization and configuration

- **SuperPlane:** Canvas update permission on the target canvas; `AGENT_HTTP_URL` must resolve
  to the agent HTTP service (in Docker Compose, use the agent service hostname, not `localhost`
  from inside the app container).
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
3. UI calls `POST /api/v1/config-assistant/suggest` with field context and instruction.
4. Go authorizes, mints `config-assistant` JWT, POSTs to agent.
5. Agent returns `{ value, explanation }`; user confirms; value is written to configuration
   and autosave/save rules apply as usual.

## Acceptance Criteria

1. Eligible fields show the assistant; **sensitive** and disallowed types do not.
2. Successful suggest returns a non-empty `value`; errors surface in the panel without
   breaking the rest of the sidebar.
3. Builder JWTs cannot be used for `/config-assistant/suggest` and vice versa (purpose
   separation).
4. User without canvas **update** cannot obtain successful suggests for that canvas.
5. Agent and app share JWT signing secret; misconfiguration yields clear 503/4xx behavior
   operable from logs.

## Risks and Mitigations

- **Risk:** Context JSON leaks secrets or PII to the model.  
  **Mitigation:** exclude sensitive fields; cap payload size; document trust boundary; add
  redaction in Go in a follow-up.

- **Risk:** Model proposes invalid expression or wrong type.  
  **Mitigation:** user must confirm; existing validation on save; prompt guidance in
  `agent/src/config_assistant/system_prompt.txt`.

- **Risk:** Cost / abuse.  
  **Mitigation:** instruction and context limits; future rate limits and org AI flags.

- **Risk:** Operator misconfigures `AGENT_HTTP_URL`.  
  **Mitigation:** document Compose defaults; log failed forwards from Go.

## Open Questions

1. Should we align **org-level “AI off”** with AI Builder so suggests return 403 when the org
   disables AI features?
2. Do we add **catalog-level** opt-in (`Field` schema) without widening blast radius, or keep
   the allowlist until product stabilizes?
3. Should we add **streaming** suggestions or multi-turn inside the panel only?
4. Do we need **structured `value`** (e.g. JSON) for future field types, or stay string-only?

## Related

- Implementation tracks work such as [issue #3714](https://github.com/superplanehq/superplane/issues/3714).
