import type { ConfigurationField } from "@/api-client";

export const EXPRESSION_DOUBLE_BRACE_TIP = "Tip: type `{{` to start an expression.";
export const EXPRESSION_PAYLOAD_TIP = "Tip: type `$` to browse node payloads.";

/** Field types that render an AutoCompleteInput expression quick tip. */
export function fieldShowsExpressionQuickTip(
  field: Pick<ConfigurationField, "type">,
  allowExpressions: boolean,
): boolean {
  if (!allowExpressions) return false;

  switch (field.type) {
    case "string":
    case "text":
    case "expression":
    case "any-predicate-list":
    case "integration-resource":
      return true;
    default:
      return false;
  }
}

export function expressionQuickTipForField(field: Pick<ConfigurationField, "type">): string {
  if (field.type === "expression") {
    return EXPRESSION_PAYLOAD_TIP;
  }
  return EXPRESSION_DOUBLE_BRACE_TIP;
}

/**
 * Resolve the AutoCompleteInput `quickTip` prop for expression-capable fields.
 * Returns `null` when the field already has a description so the tip can move
 * into a label tooltip instead of stacking under the description.
 */
export function resolveExpressionQuickTip(
  field: Pick<ConfigurationField, "type" | "description">,
  allowExpressions: boolean,
): string | null | undefined {
  if (!fieldShowsExpressionQuickTip(field, allowExpressions)) return undefined;
  if (field.description) return null;
  return expressionQuickTipForField(field);
}
