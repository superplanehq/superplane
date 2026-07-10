import type { ConfigurationField } from "@/api-client";

export type ReadonlyExpressionPreview = {
  status: "resolved" | "error";
  label: string;
  value: string;
  templateValue?: string;
};

export function buildReadonlyExpressionPreview({
  field,
  value,
  templateValue,
  errorMessage,
}: {
  field: ConfigurationField;
  value: unknown;
  templateValue?: unknown;
  context?: Record<string, unknown> | null;
  errorMessage?: string;
}): ReadonlyExpressionPreview | null {
  const templateExpressionValue = expressionTemplateValue(field, templateValue);
  const expressionValue = templateExpressionValue ?? valueExpressionValue(value);
  const fieldExpressionError = findFieldExpressionError(errorMessage, field.name, expressionValue);
  if (fieldExpressionError) {
    return {
      status: "error",
      label: "Expression error",
      value: fieldExpressionError,
      templateValue: expressionValue,
    };
  }

  if (!expressionValue) return null;

  if (!templateExpressionValue) return null;

  return {
    status: "resolved",
    label: "Applied preview",
    value: formatAppliedValue(value),
    templateValue: templateExpressionValue,
  };
}

function expressionTemplateValue(field: ConfigurationField, templateValue: unknown): string | null {
  if (typeof templateValue === "string" && (field.type === "expression" || hasExpression(templateValue))) {
    return templateValue;
  }

  return null;
}

function valueExpressionValue(value: unknown): string {
  if (typeof value === "string") return value;
  return "";
}

function formatAppliedValue(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "";
  }

  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return JSON.stringify(value, null, 2);
}

function containsWrappedExpression(value: string): boolean {
  return /\{\{[\s\S]*?\}\}/.test(value);
}

function hasExpression(value: string): boolean {
  return containsWrappedExpression(value) || value.trim().startsWith("$") || value.includes("root()");
}

function findFieldExpressionError(
  errorMessage: string | undefined,
  fieldName: string | undefined,
  expressionValue: string,
): string | null {
  if (!errorMessage) return null;

  const match = errorMessage.match(/error resolving field\s+([^:]+):\s*(.*)$/i);
  if (match) {
    const errorField = match[1].trim().replace(/^["']|["']$/g, "");
    if (fieldName && isSameField(errorField, fieldName)) {
      return formatFieldExpressionError(match[2]?.trim() || errorMessage);
    }

    const expressionError = findErrorForExpression(errorMessage, expressionValue);
    return expressionError ? formatFieldExpressionError(expressionError) : null;
  }

  const expressionError = findErrorForExpression(errorMessage, expressionValue);
  return expressionError ? formatFieldExpressionError(expressionError) : null;
}

function findErrorForExpression(errorMessage: string, expressionValue: string): string | null {
  if (!expressionValue || !errorMessage.toLowerCase().includes("expression evaluation failed")) return null;

  const failedExpression = extractFailedExpression(errorMessage);
  if (!failedExpression) return null;
  if (!normalizedExpressionValue(expressionValue).includes(normalizeExpression(failedExpression))) return null;

  return errorMessage.trim();
}

function isSameField(errorField: string, fieldName: string): boolean {
  const normalizedErrorField = normalizeFieldPath(errorField);
  const normalizedFieldName = normalizeFieldPath(fieldName);

  return (
    normalizedErrorField === normalizedFieldName ||
    normalizedErrorField.endsWith(`.${normalizedFieldName}`) ||
    normalizedErrorField.endsWith(`[${normalizedFieldName}]`)
  );
}

function normalizeFieldPath(field: string): string {
  return field
    .replace(/^["']|["']$/g, "")
    .trim()
    .toLowerCase();
}

function extractFailedExpression(errorMessage: string): string | null {
  const parts = errorMessage.split("|");
  if (parts.length < 2) return null;

  return parts[1].trim() || null;
}

function normalizedExpressionValue(value: string): string {
  return normalizeExpression(value.replace(/\{\{/g, "").replace(/\}\}/g, ""));
}

function normalizeExpression(value: string): string {
  return value.replace(/\s+/g, "");
}

function formatFieldExpressionError(errorMessage: string): string {
  return errorMessage
    .replace(/^error resolving field\s+[^:]+:\s*/i, "")
    .split("|")[0]
    .trim();
}
