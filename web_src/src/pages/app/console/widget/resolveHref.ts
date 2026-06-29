import { buildEnv, compileTemplate, evalTemplate } from "./celExpr";
import { interpolate } from "./fieldPath";

const EXPR_RE = /\{\{[\s\S]*?\}\}/;
const LEGACY_PLACEHOLDER_RE = /\{[^{}]+\}/;

/**
 * Resolve a table column's `href` template against a row. Supports three
 * surfaces, evaluated in order so authors can mix them in one string:
 *
 *   1. `{{ cel }}` expressions and templates — e.g. `{{ prUrl }}` or
 *      `https://github.com/{{ org }}/pull/{{ prNumber }}`.
 *   2. Legacy single-brace `{field}` placeholders (kept for backwards
 *      compatibility with links authored before `{{ }}` was supported).
 *   3. Bare static URLs — passed through untouched.
 *
 * Returns an empty string when `template` is missing so the cell renderer
 * can fall back to the column value.
 */
export function resolveHref(template: string | undefined, row: Record<string, unknown>): string {
  if (!template) return "";
  let working = template;
  if (EXPR_RE.test(working)) {
    const compiled = compileTemplate(working);
    if (compiled.hasExpr) {
      working = evalTemplate(compiled, row, buildEnv(), (v) => String(v ?? ""));
    }
  }
  if (LEGACY_PLACEHOLDER_RE.test(working)) {
    working = interpolate(working, row);
  }
  return working;
}
