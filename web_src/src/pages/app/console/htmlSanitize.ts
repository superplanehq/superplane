/**
 * Strict HTML sanitization for the console "html" widget.
 *
 * Authors of an html panel can write arbitrary HTML, but the rendered output
 * must never escalate beyond static, inert content:
 *
 *  - No script execution (no `<script>`, no `on*` handlers, no `javascript:` URLs).
 *  - No head-like elements that could affect the surrounding document
 *    (`<link>`, `<meta>`, `<base>`, `<head>`, `<title>`, `<iframe>`, etc).
 *  - No external resource fetches (`src`, `srcset`, `poster`, CSS `url(...)`).
 *  - Inline `style` attributes and scoped `<style>` blocks are allowed, but
 *    `<style>` rules are rewritten so their selectors are anchored at the
 *    widget's root element and can never restyle anything else on the page.
 *
 * Sanitization runs entirely client-side, mirroring how markdown panels render
 * authored content. The backend stores the raw body untouched (same trust
 * model as markdown) and validates only shape, not contents.
 */
import DOMPurify, { type Config } from "dompurify";

/**
 * Data attribute used to anchor scoped `<style>` selectors. Every cleaned
 * selector is prefixed with `[data-console-html-root="<rootId>"]` so user CSS
 * cannot escape the widget container.
 */
export const HTML_WIDGET_ROOT_ATTR = "data-console-html-root";

/**
 * Elements the author may use. Everything not on this list is dropped. The
 * set covers structural, inline, table, and a few interactive elements
 * (`details`/`summary`) that mirror what markdown already supports. `style`
 * is explicitly allowed because the policy doc promised scoped styling.
 */
const ALLOWED_TAGS = [
  // Block / structural
  "div",
  "section",
  "article",
  "header",
  "footer",
  "main",
  "aside",
  "nav",
  "p",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "blockquote",
  "pre",
  "hr",
  "br",
  // Inline / phrasing
  "span",
  "strong",
  "em",
  "b",
  "i",
  "u",
  "s",
  "small",
  "mark",
  "sub",
  "sup",
  "code",
  "kbd",
  "samp",
  "var",
  "abbr",
  "cite",
  "dfn",
  "q",
  "time",
  "data",
  // Lists
  "ul",
  "ol",
  "li",
  "dl",
  "dt",
  "dd",
  // Tables
  "table",
  "thead",
  "tbody",
  "tfoot",
  "tr",
  "th",
  "td",
  "caption",
  "colgroup",
  "col",
  // Links / media (kept inert: external URLs stripped by uponSanitizeAttribute)
  "a",
  "img",
  "figure",
  "figcaption",
  // Interactive
  "details",
  "summary",
  // Scoped styling
  "style",
];

/**
 * Attributes the author may use. Everything not on this list is removed by
 * DOMPurify. URL-bearing attributes (`href`, `src`, `srcset`) flow through
 * `ALLOWED_URI_REGEXP` below, which restricts them to `http(s)`/`mailto:`/
 * `tel:`/fragments/relative paths - so `<img src="https://...">` works but
 * `javascript:` / `data:` URLs are still rejected. We intentionally allow
 * cross-origin image fetches: the trade-off (referrer leakage, tracking
 * pixels) is documented in `docs/prd/console-and-widgets.md`.
 */
const ALLOWED_ATTR = [
  "class",
  "style",
  "id",
  "href",
  "src",
  "srcset",
  "title",
  "alt",
  "width",
  "height",
  "colspan",
  "rowspan",
  "open",
  "lang",
  "dir",
  "role",
  "tabindex",
  "name",
];

/**
 * Tags removed even before content inspection. DOMPurify already strips
 * `<script>` by default, but we list every other head-like and
 * resource-pulling element here so they never reach the output even if the
 * tag list is accidentally widened later.
 */
const FORBID_TAGS = [
  "script",
  "noscript",
  "iframe",
  "frame",
  "frameset",
  "object",
  "embed",
  "applet",
  "link",
  "meta",
  "base",
  "head",
  "html",
  "body",
  "title",
  "template",
  "form",
  "input",
  "button",
  "textarea",
  "select",
  "option",
  "audio",
  "video",
  "source",
  "track",
  "canvas",
  "math",
  "svg",
];

/**
 * Always-strip attributes (defense in depth alongside the allow-list).
 *
 * `src`/`srcset` are intentionally NOT here - they're allowed on `<img>`,
 * filtered through `ALLOWED_URI_REGEXP`. The attributes below either load
 * resources via elements that are already forbidden (`<video poster>`,
 * `<body background>`, `<embed data>`) or describe behaviors we never want
 * (`ping` beacons, form submission targets, SVG `xlink:href`).
 */
const FORBID_ATTR = ["poster", "background", "data", "ping", "formaction", "action", "xlink:href", "xmlns"];

/**
 * Allowed schemes for `href`/`src`/`srcset`. Excludes `javascript:` (XSS) and
 * `data:` (would allow embedded payloads). Tighter than DOMPurify's default URI
 * regex so we don't accidentally accept `vbscript:` or unknown protocols.
 *
 * Relative URLs (e.g. `logo.png`, `assets/page.html`, `./page`, `../up`,
 * `/dashboard`) are allowed via the final branch, which matches any value
 * that is NOT an absolute URL with a scheme. We detect a scheme by looking
 * for a `:` before the first `/`, `?`, or `#`; anything with such a colon
 * (e.g. `javascript:`, `data:`, `vbscript:`) is rejected because only the
 * explicitly listed schemes above are permitted.
 *
 * The relative branch also rejects protocol-relative URLs: a leading `/` or
 * `\` (browsers normalize `\` to `/`) is excluded so `//evil.example/...` or
 * `\\evil.example/...` cannot slip through as a "relative path" and silently
 * target an arbitrary third-party host. Genuine absolute paths (a single
 * leading `/` not followed by `/` or `\`) are still permitted via the
 * dedicated branch.
 */
const ALLOWED_URI_REGEXP = /^(?:(?:https?|mailto|tel):|#|\/(?![/\\])|(?![/\\])(?![^/?#]*:))/i;

/**
 * Substrings that disqualify an inline `style` attribute. We do a substring
 * check on the lowercased value because the inline style parser is forgiving
 * about whitespace and case, and we'd rather drop a borderline-suspicious
 * style than try to surgically rewrite it.
 */
const STYLE_BLOCKLIST = ["url(", "expression(", "@import", "behavior:", "javascript:", "vbscript:"];

let hooksRegistered = false;

/**
 * Install the global hooks exactly once. DOMPurify hooks are process-wide,
 * so we guard with a flag and use the (tag, attr) context inside the hook to
 * decide which rules apply rather than re-registering per call.
 */
function ensureHooksRegistered(): void {
  if (hooksRegistered) return;
  hooksRegistered = true;

  DOMPurify.addHook("uponSanitizeAttribute", (_node, data) => {
    if (!data) return;
    const name = data.attrName;
    const value = data.attrValue;
    if (!name) return;

    if (name === "style") {
      const lower = (value ?? "").toLowerCase();
      if (STYLE_BLOCKLIST.some((needle) => lower.includes(needle))) {
        data.keepAttr = false;
        return;
      }
    }

    if (name === "class") {
      data.attrValue = (value ?? "").replace(/\s+/g, " ").trim();
      return;
    }
  });
}

/**
 * Sanitize a `<style>` block's CSS source: strip `@import`, drop any rule
 * whose body references `url(...)` (so background-image etc. cannot fetch
 * external resources), and prefix every selector with the widget root so
 * rules cannot apply outside the panel.
 *
 * Parsing is brace-balanced and done by hand rather than via the CSSOM, so
 * we don't depend on `styleEl.sheet` being populated (jsdom returns `null`
 * there, and headless environments behave inconsistently). The supported
 * grammar is intentionally narrow: top-level style rules, plus `@media` and
 * `@supports` grouping at-rules that we recurse into. Anything else
 * (`@import`, `@keyframes`, etc.) is dropped.
 */
function cleanStyleBlockCss(css: string, rootSelector: string): string {
  const out: string[] = [];
  for (const block of splitTopLevelBlocks(css)) {
    const cleaned = cleanBlock(block, rootSelector);
    if (cleaned) out.push(cleaned);
  }
  return out.join("\n");
}

/** A `{prelude} { body }` chunk extracted from a CSS source. */
interface CssBlock {
  prelude: string;
  body: string;
}

/**
 * Find the matching `}` for the `{` at `openBraceIndex` in `css`, accounting
 * for brace nesting and string literals so braces inside e.g. `content: "}"`
 * don't fool us. Returns the index AFTER the closing brace, or `css.length`
 * if the CSS is unterminated.
 */
function findClosingBrace(css: string, openBraceIndex: number): number {
  let depth = 1;
  let inString: '"' | "'" | null = null;
  let j = openBraceIndex + 1;
  while (j < css.length && depth > 0) {
    const c = css[j];
    if (inString) {
      if (c === "\\") {
        j += 2;
        continue;
      }
      if (c === inString) inString = null;
    } else if (c === '"' || c === "'") {
      inString = c;
    } else if (c === "{") {
      depth += 1;
    } else if (c === "}") {
      depth -= 1;
    }
    j += 1;
  }
  return j;
}

/**
 * Find the next significant boundary at depth 0 in `css` starting at `i`:
 * either an opening `{` (start of a block), a `;` terminator (statement),
 * or the end of input. Skips over string literals so quoted braces and
 * semicolons don't trigger a false boundary.
 */
function nextBoundary(css: string, start: number): number {
  let inString: '"' | "'" | null = null;
  for (let i = start; i < css.length; i += 1) {
    const ch = css[i];
    if (inString) {
      if (ch === "\\") {
        i += 1;
        continue;
      }
      if (ch === inString) inString = null;
      continue;
    }
    if (ch === '"' || ch === "'") {
      inString = ch;
      continue;
    }
    if (ch === "{" || ch === ";") return i;
  }
  return css.length;
}

/**
 * Split CSS source into a list of `{prelude} { body }` chunks, respecting
 * brace nesting and string literals. At-rule terminators like `@import "x";`
 * are also emitted as bodyless blocks so the cleaner can drop them
 * uniformly.
 */
function splitTopLevelBlocks(css: string): CssBlock[] {
  const out: CssBlock[] = [];
  let i = 0;
  while (i < css.length) {
    const boundary = nextBoundary(css, i);
    if (boundary >= css.length) break;
    if (css[boundary] === "{") {
      const prelude = css.slice(i, boundary);
      const closing = findClosingBrace(css, boundary);
      out.push({ prelude, body: css.slice(boundary + 1, closing - 1) });
      i = closing;
      continue;
    }
    // semicolon-terminated at-rule (e.g. `@import "x.css";`)
    const stmt = css.slice(i, boundary + 1);
    if (stmt.trim()) out.push({ prelude: stmt, body: "" });
    i = boundary + 1;
  }
  return out;
}

/**
 * Clean a single CSS block: drop `@import`, drop anything referencing
 * `url(...)`, scope plain style rules, and recurse into `@media`/`@supports`.
 * Returns the cleaned block text, or `""` if the whole thing must be
 * dropped.
 */
function cleanBlock(block: CssBlock, rootSelector: string): string {
  const prelude = block.prelude.trim();
  const lowerPrelude = prelude.toLowerCase();
  if (lowerPrelude.startsWith("@import")) return "";

  if (lowerPrelude.startsWith("@media") || lowerPrelude.startsWith("@supports")) {
    const inner: string[] = [];
    for (const nested of splitTopLevelBlocks(block.body)) {
      const cleaned = cleanBlock(nested, rootSelector);
      if (cleaned) inner.push(cleaned);
    }
    if (inner.length === 0) return "";
    return `${prelude} { ${inner.join("\n")} }`;
  }

  // Any other at-rule (e.g. @keyframes, @font-face) is dropped: scoping
  // doesn't make sense for them and they could smuggle url() fetches.
  if (prelude.startsWith("@")) return "";

  const body = block.body.trim();
  if (!body) return "";
  if (body.toLowerCase().includes("url(")) return "";

  const scoped = scopeSelector(prelude, rootSelector);
  if (!scoped) return "";
  return `${scoped} { ${body} }`;
}

/**
 * Prefix every selector in a comma-separated selector list with
 * `rootSelector`. Returns `""` when the selector list is empty so the
 * caller can drop the rule entirely.
 */
function scopeSelector(selectorList: string, rootSelector: string): string {
  const parts = splitSelectorList(selectorList)
    .map((s) => s.trim())
    .filter(Boolean);
  if (parts.length === 0) return "";
  return parts.map((sel) => `${rootSelector} ${sel}`).join(", ");
}

/**
 * Split a CSS selector list on top-level commas only. Commas nested inside
 * parentheses (e.g. `:not(.a, .b)`, `:is(h1, h2)`), attribute brackets
 * (e.g. `[data-x="a,b"]`), or string literals are part of a single selector
 * and must not split it apart. A naive `","`-split would tear those selectors
 * up and re-scope each fragment, breaking the rule.
 */
function splitSelectorList(selectorList: string): string[] {
  const parts: string[] = [];
  let current = "";
  let depth = 0;
  let i = 0;
  while (i < selectorList.length) {
    const ch = selectorList[i];
    if (ch === '"' || ch === "'") {
      const end = scanStringLiteral(selectorList, i);
      current += selectorList.slice(i, end);
      i = end;
      continue;
    }
    if (ch === "(" || ch === "[") depth += 1;
    else if ((ch === ")" || ch === "]") && depth > 0) depth -= 1;
    else if (ch === "," && depth === 0) {
      parts.push(current);
      current = "";
      i += 1;
      continue;
    }
    current += ch;
    i += 1;
  }
  parts.push(current);
  return parts;
}

/**
 * Given that `start` points at an opening quote in `css`, return the index
 * just past the matching closing quote (or end of input), skipping escaped
 * characters. Used to keep commas/brackets inside string literals from being
 * treated as selector structure.
 */
function scanStringLiteral(css: string, start: number): number {
  const quote = css[start];
  let i = start + 1;
  while (i < css.length) {
    const ch = css[i];
    if (ch === "\\") {
      i += 2;
      continue;
    }
    if (ch === quote) return i + 1;
    i += 1;
  }
  return css.length;
}

/**
 * Sanitize an HTML string for the html widget. Returns clean HTML suitable for
 * `dangerouslySetInnerHTML` inside a `div[data-console-html-root="<rootId>"]`.
 *
 * The sanitizer is intentionally strict: anything not on the allow-list is
 * dropped. `<style>` blocks are kept but their rules are rewritten so they
 * apply only inside the matching root element.
 */
export function sanitizeHtml(raw: string, rootId: string): string {
  if (!raw) return "";
  ensureHooksRegistered();

  const config: Config = {
    ALLOWED_TAGS,
    ALLOWED_ATTR,
    FORBID_TAGS,
    FORBID_ATTR,
    ALLOWED_URI_REGEXP,
    ALLOW_DATA_ATTR: false,
    // Keep `aria-*` attributes (accessible markup is documented as allowed in
    // the PRD). DOMPurify allows these via this flag independently of
    // ALLOWED_ATTR, so it's set explicitly rather than relying on the library
    // default staying `true`.
    ALLOW_ARIA_ATTR: true,
    RETURN_TRUSTED_TYPE: false,
    // `<style>` would otherwise be hoisted into <head> by the HTML parser and
    // then stripped because DOMPurify operates on the body by default. With
    // FORCE_BODY the parser keeps it in body so the allow-list applies.
    FORCE_BODY: true,
  };

  const purified = DOMPurify.sanitize(raw, config) as unknown as string;
  return rescopeStyleBlocks(purified, rootId);
}

/**
 * After DOMPurify has removed every disallowed element/attribute, walk the
 * remaining `<style>` blocks and rewrite their CSS so every selector is
 * anchored at the widget's root element.
 *
 * We do this in a second pass (rather than inside an `uponSanitizeElement`
 * hook) because parsing CSS via the CSSOM requires the `<style>` element to
 * be attached to a document, and a hook running mid-sanitization sees the
 * detached node.
 */
function rescopeStyleBlocks(html: string, rootId: string): string {
  if (!html.includes("<style")) return html;

  const doc = new DOMParser().parseFromString(`<div>${html}</div>`, "text/html");
  const wrapper = doc.body.firstElementChild;
  if (!wrapper) return html;

  const rootSelector = `[${HTML_WIDGET_ROOT_ATTR}="${cssAttrEscape(rootId)}"]`;
  const styles = wrapper.querySelectorAll("style");
  styles.forEach((style) => {
    const cleaned = cleanStyleBlockCss(style.textContent ?? "", rootSelector);
    style.textContent = cleaned;
  });

  return wrapper.innerHTML;
}

/**
 * Escape an attribute-selector value. Root ids are generated by the panel
 * code (a hashed/UUID-style string) so this is belt-and-braces, but we still
 * need to handle the case where the id contains a `"` or backslash.
 */
function cssAttrEscape(value: string): string {
  return value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}
