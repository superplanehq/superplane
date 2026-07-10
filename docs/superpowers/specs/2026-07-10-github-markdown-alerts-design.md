# GitHub-style markdown alerts in Files and Console

**Date:** 2026-07-10  
**Status:** Approved for planning  
**Surfaces:** Files `.md` preview, Console markdown panels (shared `MarkdownContent`)

## Goal

Support GitHub Flavored Markdown alerts (`> [!NOTE]`, etc.) in both Files and Console so README-style docs and console panels render the same callouts.

## Decisions

| Topic | Choice |
| --- | --- |
| Feature | GitHub Alerts (not other “annotation” features) |
| Visual style | SuperPlane chrome (white/slate surface, thin accent bar, title-case label) — not GitHub’s tinted backgrounds |
| Types | All five: `NOTE`, `TIP`, `IMPORTANT`, `WARNING`, `CAUTION` |
| Implementation | Custom `blockquote` renderer inside `MarkdownContent` (no new remark plugin / dependency) |
| Scope | Shared renderer only; agent chat out of scope |

## Architecture

Extend `web_src/src/pages/app/Markdown.tsx` only. Files and Console already consume `MarkdownContent`, so both pick up alerts automatically.

1. Register a custom `blockquote` component on `ReactMarkdown`.
2. Inspect the blockquote’s first text line for a GitHub alert marker: `[!TYPE]` where `TYPE` is one of the five names (case-insensitive).
3. On match:
   - Render an alert shell (accent bar + label + body) instead of the default blockquote.
   - Omit the marker line from the body.
   - Keep nested markdown in the body (links, code, lists, `node:` / `integration:` chips).
4. On no match (or unknown marker such as `[!TODO]`): keep existing blockquote styles.

Styles live in a sibling module (e.g. `markdownAlertStyles.ts`) next to other markdown token modules, with light and dark variants aligned to Console/Files chrome.

Remark/rehype plugins and sanitize schema stay unchanged; alerts are still blockquotes in the AST.

## Behavior & edge cases

- Display labels: Title Case (`Note`, `Tip`, `Important`, `Warning`, `Caution`).
- Multi-paragraph content is supported when it remains one blockquote.
- Nested alerts are out of scope (GitHub also discourages nesting).
- Custom titles (`> [!NOTE] Custom title`) are out of scope for v1; treat as non-matching / plain blockquote unless the first line is exactly the marker.
- CEL interpolation in Console still runs before markdown render (unchanged).

## Testing & fixtures

- Unit tests in `Markdown.spec.tsx`:
  - Each of the five types renders as an alert (not a plain blockquote).
  - Unknown marker stays a plain blockquote.
  - Body markdown (e.g. a link or inline code) still works inside an alert.
- Update Storybook showcase README (`cleanCodeAssessment.README.md`) with examples of all five types for visual checks in Files and Console.

## Out of scope

- Agent chat / `RichMessage` pipeline
- New npm dependencies for alert parsing
- GitHub-faithful tinted alert skins
- E2E coverage for this pass
