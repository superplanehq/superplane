/**
 * Dashboard widget `show` / row filter clause evaluation.
 *
 * Clauses are bare CEL expressions (no `{{ }}` wrapper) routed through the
 * same engine as column templates (`celExpr.ts` + the builtins registered in
 * `celBuiltins.ts`), so comparisons (`==`, `!=`, `>`, `<`, `>=`, `<=`),
 * logical operators (`&&`, `||`, `!`), parentheses, arithmetic
 * (`+ - * / %`), function calls (`epochMs(...)`, `lower(...)`, …), the `in`
 * operator, and ternaries all work — one expression language everywhere.
 *
 * We never call `eval` or `new Function`: `@marcbachmann/cel-js` interprets
 * its own AST, so widgets imported from untrusted YAML can't escape the
 * sandbox.
 *
 * Sugar preserved from the retired hand-rolled evaluator:
 * - `row.status` and bare `status` both resolve against the row (a `row`
 *   binding pointing at the row itself is injected; an actual `row` field on
 *   the row takes precedence).
 * - Root identifiers absent from the row are bound to `null` instead of
 *   raising CEL's "Unknown variable", so `status == "deleted"` is simply
 *   `false` on rows without a `status` field.
 * - Any compile or eval error logs a console warning and returns
 *   `defaultValue`, so widgets remain functional even when authoring
 *   mistakes are present.
 *
 * Known divergence: equality is typed (CEL semantics) — the old evaluator's
 * loose scalar normalization made `10 == "10"` and `true == "true"` true;
 * they are now false. Ordering comparisons on stringified rows still work
 * via `evalExprDetailed`'s numeric-string retry.
 */

import { buildEnv, compileExpr, type CompiledExpr, evalExprDetailed } from "./celExpr";

interface CompiledShow {
  expr: CompiledExpr;
  roots: string[];
}

/**
 * Compile cache: filters re-evaluate the same clause once per row, and
 * `Environment.parse` is the expensive step. Cleared wholesale at a size
 * bound so transient strings typed in the widget editor can't grow it
 * forever.
 */
const COMPILED = new Map<string, CompiledShow>();
const COMPILED_CACHE_LIMIT = 500;

function compiledFor(source: string): CompiledShow {
  let entry = COMPILED.get(source);
  if (!entry) {
    if (COMPILED.size >= COMPILED_CACHE_LIMIT) COMPILED.clear();
    entry = { expr: compileExpr(source), roots: rootIdentifiers(source) };
    COMPILED.set(source, entry);
  }
  return entry;
}

const IDENT_START = /[A-Za-z_]/;
const IDENT_CHAR = /[A-Za-z0-9_]/;

/**
 * Names that must never be null-bound as "missing row fields": CEL keywords
 * and literals, plus the adapter-provided `row` alias and the `now` global
 * from `buildEnv`.
 */
const RESERVED_IDENTIFIERS = new Set(["true", "false", "null", "in", "row", "now"]);

/**
 * Collect the root identifiers referenced by `source` — identifier tokens
 * outside string literals that are not preceded by `.` (those are field
 * selections, not roots). Used to bind missing row fields to `null` so a
 * clause over a heterogeneous row set degrades to `false` comparisons
 * instead of "Unknown variable" errors. Function names may be collected
 * too; binding them is harmless because CEL resolves calls through the
 * function registry, not the variable context.
 */
function rootIdentifiers(source: string): string[] {
  const roots = new Set<string>();
  let i = 0;
  while (i < source.length) {
    const ch = source[i];
    if (ch === '"' || ch === "'") {
      i = skipStringLiteral(source, i, ch);
      continue;
    }
    if (IDENT_START.test(ch)) {
      let j = i + 1;
      while (j < source.length && IDENT_CHAR.test(source[j])) j++;
      const name = source.slice(i, j);
      const isRawStringPrefix = name === "r" && (source[j] === '"' || source[j] === "'");
      if (source[i - 1] !== "." && !isRawStringPrefix && !RESERVED_IDENTIFIERS.has(name)) {
        roots.add(name);
      }
      i = j;
      continue;
    }
    i++;
  }
  return [...roots];
}

/** Skip past a quoted string literal (honoring backslash escapes). */
function skipStringLiteral(source: string, start: number, quote: string): number {
  let i = start + 1;
  while (i < source.length && source[i] !== quote) {
    i += source[i] === "\\" ? 2 : 1;
  }
  return i + 1;
}

/**
 * Evaluate the given expression against a row context. Returns a boolean
 * (the truthiness of the resulting value). On any compile or eval error
 * this logs to console and returns `defaultValue` (defaults to `true`), so
 * widgets remain functional even when authoring mistakes are present.
 */
export function evaluateShow(expression: string | undefined, row: unknown, defaultValue = true): boolean {
  if (!expression || !expression.trim()) return defaultValue;
  const { expr, roots } = compiledFor(expression.trim());
  if (!expr.ok) return warnAndDefault(expr.error, defaultValue);

  const record = row && typeof row === "object" && !Array.isArray(row) ? (row as Record<string, unknown>) : {};
  const globals: Record<string, unknown> = { row: record };
  for (const name of roots) {
    if (!Object.prototype.hasOwnProperty.call(record, name)) globals[name] = null;
  }

  const result = evalExprDetailed(expr, record, buildEnv(globals));
  if (!result.ok) return warnAndDefault(result.error, defaultValue);
  return Boolean(result.value);
}

function warnAndDefault(message: string, defaultValue: boolean): boolean {
  if (typeof console !== "undefined") {
    console.warn(`Dashboard widget expression failed: ${message}`);
  }
  return defaultValue;
}
