import { buildEnv, compileMaybeExpr, evalBoolExpr, evalExpr } from "./celExpr";
import { evaluateShow } from "./showExpression";

const LEGACY_SHOW_RE = /^([A-Za-z_][\w.]*)\s*==\s*(?:"([^"]*)"|'([^']*)')$/;

/**
 * Evaluate a row `show` clause: CEL `{{ }}`, legacy `field == "value"`, or
 * the dashboard mini expression language (`row.status == "failed"`).
 */
export function evaluateRowShow(expression: string | undefined, row: unknown, defaultValue = true): boolean {
  if (!expression || !expression.trim()) return defaultValue;
  const trimmed = expression.trim();
  const record = row && typeof row === "object" && !Array.isArray(row) ? (row as Record<string, unknown>) : {};

  if (trimmed.includes("{{")) {
    const env = buildEnv();
    if (/^\s*\{\{[\s\S]*\}\}\s*$/.test(trimmed)) {
      const maybe = compileMaybeExpr(trimmed);
      if (maybe.kind === "expr") {
        return Boolean(evalExpr(maybe.expr, record, env));
      }
    }
    return evalBoolExpr(trimmed, record, env);
  }

  const legacy = trimmed.match(LEGACY_SHOW_RE);
  if (legacy) {
    const field = legacy[1]!;
    const expected = legacy[2] ?? legacy[3] ?? "";
    const actual = record[field];
    return String(actual ?? "") === expected;
  }

  return evaluateShow(trimmed, row, defaultValue);
}
