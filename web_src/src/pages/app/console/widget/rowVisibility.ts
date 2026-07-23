import { buildEnv, compileExpr, compileMaybeExpr, evalBoolExpr, evalExpr, evalExprDetailed } from "./celExpr";
import { tryEvaluateShow } from "./showExpression";

const LEGACY_SHOW_RE = /^([A-Za-z_][\w.]*)\s*==\s*(?:"([^"]*)"|'([^']*)')$/;

/**
 * Evaluate a row `show` / widget `filter` clause against a row.
 *
 * Resolution order (first match wins) — designed so existing filters keep
 * their exact semantics while CEL-style expressions "just work":
 *  1. `{{ CEL }}` template / expression → full CEL via `celExpr`.
 *  2. Simple legacy `field == "value"` → direct string equality.
 *  3. Dashboard mini expression (`row.status == "failed"`, `count > 5`) →
 *     the sandboxed tokenizer in `showExpression`.
 *  4. Bare CEL expression the mini tokenizer can't parse (arithmetic,
 *     function calls — e.g. `epochMs(createdAt) > (float(now) - 604800.0) * 1000.0`)
 *     → evaluated as CEL without requiring `{{ }}` braces.
 *  5. Nothing parses → log once and return `defaultValue`.
 */
export function evaluateRowShow(expression: string | undefined, row: unknown, defaultValue = true): boolean {
  if (!expression || !expression.trim()) return defaultValue;
  const trimmed = expression.trim();
  const record = row && typeof row === "object" && !Array.isArray(row) ? (row as Record<string, unknown>) : {};

  if (trimmed.includes("{{")) return evalCelTemplate(trimmed, record);

  const legacyEquality = matchLegacyEquality(trimmed, record);
  if (legacyEquality !== undefined) return legacyEquality;

  // Prefer the legacy mini tokenizer so `row.`-prefixed / bare-field filters
  // keep their existing (loose-equality) semantics. Only when it genuinely
  // cannot parse the expression do we fall back to CEL — this is what lets
  // arithmetic + builtin-function filters evaluate instead of throwing.
  const legacyResult = tryEvaluateShow(trimmed, row);
  if (legacyResult.ok) return legacyResult.value;

  const bareCel = evalBareCel(trimmed, record);
  if (bareCel !== undefined) return bareCel;

  if (typeof console !== "undefined") {
    console.warn(`Dashboard widget expression failed: ${legacyResult.error}`);
  }
  return defaultValue;
}

/** Evaluate a `{{ }}` CEL template / expression against the row. */
function evalCelTemplate(trimmed: string, record: Record<string, unknown>): boolean {
  const env = buildEnv();
  if (/^\s*\{\{[\s\S]*\}\}\s*$/.test(trimmed)) {
    const maybe = compileMaybeExpr(trimmed);
    if (maybe.kind === "expr") return Boolean(evalExpr(maybe.expr, record, env));
  }
  return evalBoolExpr(trimmed, record, env);
}

/**
 * Match a simple legacy `field == "value"` equality, returning the boolean
 * result or `undefined` when the expression isn't that shape.
 */
function matchLegacyEquality(trimmed: string, record: Record<string, unknown>): boolean | undefined {
  const legacy = trimmed.match(LEGACY_SHOW_RE);
  if (!legacy) return undefined;
  const field = legacy[1]!;
  const expected = legacy[2] ?? legacy[3] ?? "";
  return String(record[field] ?? "") === expected;
}

/**
 * Evaluate a bare (unbraced) expression as CEL. Returns the boolean result, or
 * `undefined` when the expression can't be compiled/evaluated as CEL so the
 * caller can fall back to its default.
 */
function evalBareCel(trimmed: string, record: Record<string, unknown>): boolean | undefined {
  const compiled = compileExpr(trimmed);
  if (!compiled.ok) return undefined;
  const detailed = evalExprDetailed(compiled, record, buildEnv());
  return detailed.ok ? Boolean(detailed.value) : undefined;
}
