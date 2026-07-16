// Adapter contract for an expression dialect (expr-lang / CEL). The
// shared editor UI stays fixed and delegates language semantics to the
// adapter.

export type ExpressionDialectId = "expr-lang" | "cel";

export type ExpressionEvaluationOutcome = {
  ok: boolean;
  value?: unknown;
  formattedValue?: string;
  error?: string;
};

export interface ExpressionAdapter {
  id: ExpressionDialectId;
  // Full evaluation; errors surface as `ok: false` so the editor can show
  // an inline diagnostic without crashing.
  evaluate(expression: string, globals: Record<string, unknown> | null | undefined): ExpressionEvaluationOutcome;
  // Cheap resolution for the highlighted suggestion preview. Returns
  // `undefined` when the tail path isn't statically resolvable.
  resolveSuggestionValue(expression: string, globals: Record<string, unknown> | null | undefined): unknown;
  formatResult(value: unknown): string;
}
