import { evaluateExpr, formatExprResult } from "@/lib/exprEvaluator";
import { resolveSuggestionPath } from "./pathWalker";
import type { ExpressionAdapter, ExpressionEvaluationOutcome } from "./types";

// Rewrite `root()` / `previous(depth)` autocomplete shorthands into the
// keys emitted by `buildAutocompleteExampleObj` (`__root`,
// `__previousByDepth`). Returns `null` for unsafe input (e.g. `previous(x)`).
function rewriteShorthand(expr: string): string | null {
  const rootMatch = expr.match(/^root\(\)/);
  if (rootMatch) return `__root${expr.slice(rootMatch[0].length)}`;

  const previousMatch = expr.match(/^previous\(([^)]*)\)/);
  if (previousMatch) {
    const raw = (previousMatch[1] ?? "").trim();
    const depth = raw === "" ? 1 : Number(raw);
    if (!Number.isInteger(depth) || depth < 1) return null;
    return `__previousByDepth["${depth}"]${expr.slice(previousMatch[0].length)}`;
  }

  const dollarBracket = expr.startsWith("$[") ? "$" + expr.slice(1) : expr;
  return dollarBracket;
}

export function resolveExprLangSuggestionValue(
  expression: string,
  globals: Record<string, unknown> | null | undefined,
): unknown {
  return resolveSuggestionPath(expression, globals, {
    rewrite: rewriteShorthand,
    resolveRoot: (ident, g) => (ident === "$" || ident === "$env" ? (g ?? undefined) : g?.[ident]),
  });
}

export function evaluateExprLang(
  expression: string,
  globals: Record<string, unknown> | null | undefined,
): ExpressionEvaluationOutcome {
  if (!globals) return { ok: false, error: "No context available" };
  try {
    const value = evaluateExpr(expression.trim(), globals);
    return { ok: true, value, formattedValue: formatExprResult(value) };
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : "Evaluation failed" };
  }
}

export const exprLangAdapter: ExpressionAdapter = {
  id: "expr-lang",
  evaluate: evaluateExprLang,
  resolveSuggestionValue: resolveExprLangSuggestionValue,
  formatResult: formatExprResult,
};
