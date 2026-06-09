import type { ConfigurationField } from "@/api-client";
import { isUrl } from "@/lib/utils";
import type { ConfigurationDisplayKind } from "./types";

export const EMPTY_DISPLAY_VALUE = "—";

export type FormattedConfigurationValue = {
  kind: ConfigurationDisplayKind;
  displayText: string;
  href?: string;
  chips?: string[];
};

function isEmptyValue(value: unknown): boolean {
  if (value === null || value === undefined) {
    return true;
  }
  if (typeof value === "string") {
    return value.trim() === "";
  }
  if (Array.isArray(value)) {
    return value.length === 0;
  }
  if (typeof value === "object") {
    return Object.keys(value).length === 0;
  }
  return false;
}

function resolveSelectLabel(field: ConfigurationField, value: string): string {
  const options = field.typeOptions?.select?.options ?? [];
  const match = options.find((option) => option.value === value);
  return match?.label ?? value;
}

function resolveMultiSelectLabels(field: ConfigurationField, values: string[]): string[] {
  const options = field.typeOptions?.multiSelect?.options ?? [];
  return values.map((value) => {
    const match = options.find((option) => option.value === value);
    return match?.label ?? value;
  });
}

function formatSecretKeyValue(value: unknown): string {
  if (!value || typeof value !== "object") {
    return EMPTY_DISPLAY_VALUE;
  }
  const record = value as { secret?: string; key?: string };
  if (!record.secret || !record.key) {
    return EMPTY_DISPLAY_VALUE;
  }
  return `secret:${record.secret} / ${record.key}`;
}

function formatPrimitiveValue(value: unknown, field: ConfigurationField): FormattedConfigurationValue {
  if (isEmptyValue(value)) {
    return { kind: "empty", displayText: EMPTY_DISPLAY_VALUE };
  }

  if (field.type === "boolean") {
    return {
      kind: "boolean",
      displayText: value === true || value === "true" ? "Yes" : "No",
    };
  }

  if (field.type === "select") {
    return {
      kind: "text",
      displayText: resolveSelectLabel(field, String(value)),
    };
  }

  if (field.type === "multi-select" || field.type === "days-of-week") {
    const values = Array.isArray(value) ? value.map(String) : [String(value)];
    const labels = field.type === "multi-select" ? resolveMultiSelectLabels(field, values) : values;
    return {
      kind: "list",
      displayText: labels.join(", "),
      chips: labels,
    };
  }

  if (field.type === "secret-key") {
    return {
      kind: "text",
      displayText: formatSecretKeyValue(value),
    };
  }

  const stringValue = typeof value === "object" ? JSON.stringify(value, null, 2) : String(value);

  if (field.sensitive && stringValue.trim() !== "") {
    return { kind: "text", displayText: "••••••" };
  }

  if (field.type === "url" || isUrl(stringValue)) {
    return {
      kind: "url",
      displayText: stringValue,
      href: stringValue,
    };
  }

  if (field.type === "expression") {
    return {
      kind: "expression",
      displayText: stringValue,
    };
  }

  if (field.type === "text" || field.type === "xml" || field.type === "object") {
    const isMultiline = stringValue.includes("\n") || stringValue.length > 80;
    return {
      kind: isMultiline ? "code" : "text",
      displayText: stringValue,
    };
  }

  if (Array.isArray(value)) {
    const labels = value.map((item) => (typeof item === "object" ? JSON.stringify(item) : String(item)));
    return {
      kind: "list",
      displayText: labels.join(", "),
      chips: labels,
    };
  }

  return {
    kind: "text",
    displayText: stringValue,
  };
}

export function formatConfigurationValue(field: ConfigurationField, value: unknown): FormattedConfigurationValue {
  return formatPrimitiveValue(value, field);
}

export function formatConfigurationLabel(field: ConfigurationField): string {
  return field.label?.trim() || field.name || "Field";
}
