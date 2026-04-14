# Enriched report markdown — portable implementation spec

Short reference for re-implementing the same “report markdown” behavior outside this repo. Author-facing syntax is the same as in [report-markdown.md](./report-markdown.md); this file focuses on **behavior contract** and **implementation notes**.

## Rendering pipeline

| Layer | Role |
|-------|------|
| **Markdown** | GitHub Flavored Markdown (tables, strikethrough, task lists, autolinks) via `remark-gfm` (or equivalent). |
| **HTML passthrough** | Allow raw HTML for `<details>` / `<summary>` (and other safe tags) via a sanitizer that **allows** `details` and `summary` in addition to the default safe set. |
| **Sanitize** | Strip dangerous HTML; keep semantic tags needed for collapsibles. |
| **Highlight** | Fenced code blocks with a language tag get syntax highlighting (e.g. highlight.js). |

Typical plugin order (React): `remarkGfm` → `rehypeRaw` → `rehypeSanitize(schema)` → `rehypeHighlight`.

## Extensions (author syntax → behavior)

### 1. Inline badges (Option C)

- **Syntax:** Single backticks containing `type:label` with **no spaces** around the colon in the type prefix.
- **Regex (concept):** `^(status|success|warning|error|info|duration):(.+)$` on the **full** inline code text.
- **Match:** Render a **pill/badge** (not monospace code). Show **`label`** only in the pill (the `type` selects color).
- **No match:** Render as normal inline `<code>`.

| `type` | Intended use |
|--------|----------------|
| `status`, `success` | Positive / OK |
| `warning` | Warning |
| `error` | Error / failure |
| `info` | Neutral info |
| `duration` | Timing |

### 2. GitHub-style admonitions

- **Syntax:** Blockquote whose **content** includes `[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, or `[!CAUTION]` (case as shown). Often written as:

  ```markdown
  > [!WARNING]
  > Line one of body
  ```

- **Detection:** Search combined text of blockquote children for `\[! (NOTE|TIP|IMPORTANT|WARNING|CAUTION) \]` (the tag may appear **not** only at line start — parsers differ).
- **Rendering:** Replace the blockquote with a callout: left border, tinted background, icon + title (`Note`, `Tip`, …), body below. **Strip** the `[!TYPE]` token from displayed body text (including leading whitespace/newlines after strip).
- **Fallback:** Blockquotes without a recognized tag render as a normal styled blockquote.

### 3. Collapsible sections

- **Syntax:** Standard HTML:

  ```html
  <details>
  <summary>Title</summary>

  Body (can be multiple blocks)

  </details>
  ```

- **Rendering:** Style `<details>` with border/background; `<summary>` as a clickable row; body in a **separate** wrapper with horizontal padding and `white-space: pre-wrap` so preformatted / multi-line content does not collapse to one line.

### 4. Polished defaults (optional but recommended)

Override default elements for consistent UI:

| Element | Behavior |
|---------|----------|
| **Links** | `target="_blank"`, `rel="noopener noreferrer"`, optional external-link icon after label. |
| **Tables** | Wrap in horizontal scroll container; bordered cells; header row background. |
| **Images** | `max-height`, rounded corners, border (avoid huge images breaking layout). |
| **Horizontal rule** | Subtle divider spacing/color. |
| **Fenced code** | Language-tagged blocks → syntax highlight; inline code uses badge logic above when pattern matches. |

### 5. Standard GFM

Tables, task lists (`- [ ]` / `- [x]`), and strikethrough work if GFM is enabled.

## Dependencies (reference stack)

- `react-markdown`, `remark-gfm`, `rehype-raw`, `rehype-sanitize`, `rehype-highlight`, plus a highlight.js theme CSS for code blocks.

## Not in scope here

- Expression `{{ ... }}` resolution (SuperPlane-specific).
- Mermaid or other embeds — not part of this markdown extension set unless added separately.
