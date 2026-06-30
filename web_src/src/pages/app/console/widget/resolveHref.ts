import { buildEnv, compileTemplate, evalExpr } from "./celExpr";
import { interpolate } from "./fieldPath";

const LEGACY_PLACEHOLDER_RE = /\{[^{}]+\}/;

/**
 * Resolve a table column's `href` template against a row. Supports three
 * surfaces, evaluated so authors can mix them in one string:
 *
 *   1. `{{ cel }}` expressions and templates — e.g. `{{ prUrl }}` or
 *      `https://github.com/{{ org }}/pull/{{ prNumber }}`.
 *   2. Legacy single-brace `{field}` placeholders (kept for backwards
 *      compatibility with links authored before `{{ }}` was supported).
 *   3. Bare static URLs — passed through untouched.
 *
 * Legacy `{field}` interpolation is applied only to the template's literal
 * text, never to the result of a `{{ cel }}` expression. Otherwise a CEL
 * result that happens to contain literal `{...}` segments (e.g. a templated
 * or encoded URL) would be re-substituted from the row and corrupt the href.
 *
 * Returns an empty string when `template` is missing so the cell renderer
 * can fall back to the column value.
 */
export function resolveHref(template: string | undefined, row: Record<string, unknown>): string {
  if (!template) return "";
  const env = buildEnv();
  let out = "";
  for (const segment of compileTemplate(template).segments) {
    if (segment.kind === "literal") {
      out += LEGACY_PLACEHOLDER_RE.test(segment.value) ? interpolate(segment.value, row) : segment.value;
      continue;
    }
    const value = evalExpr(segment.expr, row, env);
    if (value === undefined) continue;
    out += String(value ?? "");
  }
  return out;
}
