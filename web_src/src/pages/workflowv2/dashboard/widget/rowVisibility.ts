import { buildEnv, compileExpr, compileMaybeExpr, evalBoolExpr, evalExpr } from "./celExpr";
import { evaluateShow, tryEvaluateShow } from "./showExpression";

const LEGACY_SHOW_RE = /^([A-Za-z_][\w.]*)\s*==\s*(?:"([^"]*)"|'([^']*)')$/;

/**
 * Evaluate a row `show` clause: CEL `{{ }}`, legacy `field == "value"`, or
 * the dashboard mini expression language (`row.status == "failed"`).
 *
 * Bare expressions that the mini-eval can't parse (e.g. CEL-style
 * `int(started_at) >= now() - 604800` with function calls or arithmetic) are
 * transparently retried against the CEL evaluator before falling back to the
 * provided default. This lets authors mix CEL features into widget filters
 * without forcing them to wrap every expression in `{{ … }}`.
 */
export function evaluateRowShow(expression: string | undefined, row: unknown, defaultValue = true): boolean {
  if (!expression || !expression.trim()) return defaultValue;
  const trimmed = expression.trim();
  const record = toRecord(row);

  if (trimmed.includes("{{")) {
    return evaluateCelTemplate(trimmed, record);
  }

  const legacy = trimmed.match(LEGACY_SHOW_RE);
  if (legacy) {
    const expected = legacy[2] ?? legacy[3] ?? "";
    return String(record[legacy[1]!] ?? "") === expected;
  }

  const mini = tryEvaluateShow(trimmed, row);
  if (mini.ok) return mini.value;

  const celResult = tryEvaluateCelBare(trimmed, record);
  if (celResult !== undefined) return celResult;

  return evaluateShow(trimmed, row, defaultValue);
}

function toRecord(row: unknown): Record<string, unknown> {
  if (!row || typeof row !== "object" || Array.isArray(row)) return {};
  return row as Record<string, unknown>;
}

function evaluateCelTemplate(trimmed: string, record: Record<string, unknown>): boolean {
  const env = buildEnv();
  if (/^\s*\{\{[\s\S]*\}\}\s*$/.test(trimmed)) {
    const maybe = compileMaybeExpr(trimmed);
    if (maybe.kind === "expr") {
      return Boolean(evalExpr(maybe.expr, record, env));
    }
  }
  return evalBoolExpr(trimmed, record, env);
}

function tryEvaluateCelBare(expression: string, record: Record<string, unknown>): boolean | undefined {
  const compiled = compileExpr(expression);
  if (!compiled.ok) return undefined;
  try {
    return Boolean(evalExpr(compiled, record, buildEnv()));
  } catch {
    return undefined;
  }
}
