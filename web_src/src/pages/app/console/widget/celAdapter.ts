import type { ExpressionAdapter, ExpressionEvaluationOutcome } from "@/lib/expression";
import { resolveSuggestionPath } from "@/lib/expression/pathWalker";

import { stringifyCelValue } from "./celBuiltins";
import {
  buildEnv,
  compileExpr,
  compileTemplate,
  DOLLAR_REWRITE_IDENTIFIER,
  evalExprDetailed,
  evalTemplateDetailed,
} from "./celExpr";

const FULL_TEMPLATE_RE = /^\s*\{\{[\s\S]*\}\}\s*$/;
const ANY_TEMPLATE_RE = /\{\{[\s\S]*?\}\}/;

// Sample rows synthesized for authoring previews don't know about the
// internal `__runNodes__` identifier that `$` gets rewritten to; without
// this shim the evaluator would bail with "Unknown variable: __runNodes__"
// instead of accurately reporting the missing node key. Live rows already
// carry a `__runNodes__` map so this early-returns for them.
function withRunNodesShim(row: Record<string, unknown>): Record<string, unknown> {
  if (DOLLAR_REWRITE_IDENTIFIER in row) return row;
  const existingDollar = row.$;
  const shim = existingDollar && typeof existingDollar === "object" ? existingDollar : {};
  return { ...row, [DOLLAR_REWRITE_IDENTIFIER]: shim };
}

export function evaluateCel(
  expression: string,
  globals: Record<string, unknown> | null | undefined,
): ExpressionEvaluationOutcome {
  if (!globals) return { ok: false, error: "No context available" };
  const trimmed = expression.trim();
  if (!trimmed) return { ok: true, value: "", formattedValue: "" };

  const row = withRunNodesShim(globals as Record<string, unknown>);
  const env = buildEnv();

  if (ANY_TEMPLATE_RE.test(trimmed) && !FULL_TEMPLATE_RE.test(trimmed)) {
    const outcome = evalTemplateDetailed(compileTemplate(trimmed), row, env, stringifyCelValue);
    if (!outcome.ok) return { ok: false, error: outcome.error };
    return { ok: true, value: outcome.value, formattedValue: outcome.value };
  }

  const inner = FULL_TEMPLATE_RE.test(trimmed) ? trimmed.replace(/^\s*\{\{/, "").replace(/\}\}\s*$/, "") : trimmed;
  const outcome = evalExprDetailed(compileExpr(inner), row, env);
  if (!outcome.ok) return { ok: false, error: outcome.error };
  return { ok: true, value: outcome.value, formattedValue: stringifyCelValue(outcome.value) };
}

export function resolveCelSuggestionValue(
  expression: string,
  globals: Record<string, unknown> | null | undefined,
): unknown {
  return resolveSuggestionPath(expression, globals, {
    rewrite: (expr) => {
      if (expr.startsWith("$[")) return DOLLAR_REWRITE_IDENTIFIER + expr.slice(1);
      if (expr === "$") return DOLLAR_REWRITE_IDENTIFIER;
      return expr;
    },
    resolveRoot: (ident, g) => g?.[ident],
  });
}

export const widgetCelAdapter: ExpressionAdapter = {
  id: "cel",
  evaluate: evaluateCel,
  resolveSuggestionValue: resolveCelSuggestionValue,
  formatResult: stringifyCelValue,
};
