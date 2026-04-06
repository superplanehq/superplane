# AI Builder: @-mentions for canvas nodes

## Overview

Users **@-mention canvas nodes by the same human-readable names** they see on the canvas. They **never type or see node ids** in the composer or chat. Behind the scenes, the client and agent map those mentions to **stable node ids** for proposals and expansion. Implementation spans the **web composer** (name-first autocomplete + send-time encoding) and the **Python agent** (parse id tokens, expand from **`describe_canvas`**, reuse **`canvas_cache`** when practical).

## Problem Statement

The model usually learns the graph by calling **`get_canvas`**, but users often mean **one or two specific steps**. Describing nodes in prose is ambiguous; **canvas display names** are already the mental model users use. Mentions should match that model while still giving the assistant **deterministic ids**.

## Goals

1. Type **`@`** then **letters** to filter nodes; the list shows **canvas names** (primary label), not ids.
2. **Pick a row** (keyboard or pointer); the composer inserts **`@` + that canvas name** (readable text only).
3. On **send**, the client **transcodes** readable mentions into a **stable wire form** (e.g. `@[node:<nodeId>]`) in the request body so the agent never depends on fuzzy name parsing over the wire.
4. **Pre-expand** on the agent into a short appendix (id, name, type, block name from **`describe_canvas`**; optional capped events later).
5. Keep **chat titles** sane when the expanded appendix is long (**display question vs `model_prompt`** in persistence).

## Non-Goals (v1)

- **#provider** or other sigils for catalog filtering.
- **@component** / **@trigger** / **@integration** catalog mentions (follow-ons).
- **Contenteditable “chips”** are not required; plain text + `@CanvasName` in the composer is enough if transcoding at send is correct.
- New public HTTP fields **optional** later (`mentioned_node_ids`); v1 can rely on encoded tokens inside `question`.

## User stories

1. As a builder, I type **`@`** and see nodes listed by **the names on my canvas**, then pick one without ever seeing a uuid.
2. As a builder, I can mention **several** nodes in one message; each appears as **`@Name`** in the thread.
3. As a builder, if two nodes share a name, I can still pick the right one using **secondary line** context (e.g. block / trigger id).

## Functional requirements

### Composer (web)

- Feed the picker from **`canvasNodes`** (or workflow nodes): **display name** = canvas node name (human-readable); **value** = node id used only for transcoding.
- **`@` opens** mention mode; typing after `@` **filters** by display name (case-insensitive, substring).
- **Rows**: primary = **canvas name** only; **secondary** = disambiguator when useful (e.g. `github.postMessage`, “Trigger”, “Action”) so duplicate titles remain distinguishable.
- **Insert**: after accept, insert **`@` + exact display name** (match the string used on the canvas for that node) and a trailing space.
- **Send**: replace each accepted mention with **`@[node:<id>]`** (or agreed token) in the payload the agent receives; the **UI may keep showing** `@Name` in the local bubble for that turn if the product stores display text separately—otherwise store what the user saw and let the agent parse names only when transcoding is guaranteed (prefer **always** send ids on the wire).
- **Keyboard**: arrows + Enter to accept; Enter does not send while the mention menu is open.

### Duplicate names

- If multiple nodes share the same **canvas name**, the list must still identify **one row per node** (secondary text, stable ordering). Picking a row binds **that** node’s id at transcoding time—no ambiguous server-side name resolution for v1.

### Agent (Python)

- **Parse** `@[node:…]` (or chosen wire form) from `question` after client transcoding.
- **Resolve** ids with **`describe_canvas`**; unknown ids noted in the appendix.
- Build **`model_prompt`** with original logical content; appendix lists id + name + type + block name.
- **`system_prompt`**: wire tokens are the user’s explicit node selection; appendix is authoritative.

### Persistence

- **Chat title / preview**: prefer the **human-visible** question (before or after transcoding—product choice: usually **what the user typed/saw**, without the internal appendix).
- **Model history**: use **`model_prompt`** so replay matches reasoning (see **PersistedRunRecorder** split in implementation).

## UX

- Placeholder or hint: **“@ to mention a step”** (short).
- After canvas edits, if a transcoded id is stale, expansion says the node is missing; user re-mentions from the updated list.

## Acceptance criteria

1. **`@`** opens a list labeled with **canvas names**, not ids.
2. After pick, the composer shows **`@CanvasName`**, not a uuid.
3. The **stream request** contains **id-backed** tokens the agent can parse (transcoding on send).
4. Expansion includes **correct id ↔ name** rows from **`describe_canvas`**.
5. **Duplicate titles**: two nodes with the same name can still be chosen unambiguously from the list.
6. Unit tests: transcoding / parsing and expansion with mocked **`describe_canvas`**.

## Future additions

- **`#provider`** for catalog list hints.
- **Structured `mentions`** in JSON body alongside display `question` for cleaner persistence.
- **Free-text finish**: if the filter narrows to **one** row, Enter inserts it without arrow keys (optional polish).

## Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Name changed on canvas after mention | Transcoding used id from pick time; expansion uses current `describe_canvas` (name may differ; id stable). |
| Duplicate display names | Secondary line + one row per node; no fuzzy id guess from name on server in v1. |
| Display name contains `@` or special chars | Define transcoding rules (e.g. match longest display names first, or delimiter boundaries). |
| Prompt length | Cap mentions / appendix size. |

## Open questions

1. Maximum mentions per message and appendix size?
2. Persisted **user bubble** text: store **display** (`@Name`) only, or store wire form for replay consistency? (If display-only, ensure list-messages API and agent history stay aligned with implementation.)
