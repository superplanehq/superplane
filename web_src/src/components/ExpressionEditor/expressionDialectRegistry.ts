// Dialect → adapter registry. Kept in its own module to keep the React
// component file fast-refresh clean. The CEL entry is registered lazily by
// the widget module so `@/components` doesn't import from `@/pages`.
import type { ExpressionAdapter } from "@/lib/expression";
import { exprLangAdapter } from "@/lib/expression";
import type { ExpressionEditorDialect } from "./types";

const ADAPTERS: Partial<Record<ExpressionEditorDialect, ExpressionAdapter>> = {
  "expr-lang": exprLangAdapter,
};

export function registerExpressionDialect(dialect: ExpressionEditorDialect, adapter: ExpressionAdapter): void {
  ADAPTERS[dialect] = adapter;
}

export function getExpressionDialectAdapter(dialect: ExpressionEditorDialect): ExpressionAdapter | undefined {
  return ADAPTERS[dialect];
}
