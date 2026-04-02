import type { ConfigurationField, ConfigurationSelectOption } from "@/api-client";

/**
 * Field types that may show the inline config assistant (frontend allowlist only).
 * Excludes select / multi-select (fixed catalogs), number / boolean (poor UX for NL assist).
 */
export const CONFIG_ASSISTANT_SUPPORTED_TYPES = [
  "expression",
  "string",
  "text",
  "url",
  "cron",
] as const;

export type ConfigAssistantSupportedFieldType = (typeof CONFIG_ASSISTANT_SUPPORTED_TYPES)[number];

const SUPPORTED = new Set<string>(CONFIG_ASSISTANT_SUPPORTED_TYPES);

/**
 * Single integration-resource fields can use Fixed vs Expression mode; assist only that shape (not multi pickers).
 */
function isIntegrationResourceExpressionCapable(field: ConfigurationField): boolean {
  if (field.type !== "integration-resource") {
    return false;
  }
  return field.typeOptions?.resource?.multi !== true;
}

/**
 * Whether the inline config assistant may be used for this field.
 * Sensitive fields are never assisted (no API call, no sparkle).
 */
export function isConfigAssistantSupportedField(field: ConfigurationField): boolean {
  if (field.sensitive === true) {
    return false;
  }
  const t = field.type;
  if (!t) {
    return false;
  }
  if (isIntegrationResourceExpressionCapable(field)) {
    return true;
  }
  return SUPPORTED.has(t);
}

/**
 * Parse assistant output into multi-select values: JSON array string or comma-separated.
 */
export function parseAssistantMultiSelectValue(raw: string): string[] {
  const trimmed = raw.trim();
  if (!trimmed) {
    return [];
  }
  if (trimmed.startsWith("[")) {
    try {
      const parsed = JSON.parse(trimmed) as unknown;
      if (Array.isArray(parsed)) {
        return parsed.map((v) => String(v).trim()).filter(Boolean);
      }
    } catch {
      /* fall through */
    }
  }
  return trimmed
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

export function filterToAllowedMultiSelectValues(values: string[], allowed: Set<string>): string[] {
  return values.filter((v) => allowed.has(v));
}

export function parseAssistantNumberValue(
  raw: string,
  min?: number,
  max?: number,
): { ok: true; value: number } | { ok: false; error: string } {
  const n = Number(raw.trim());
  if (Number.isNaN(n) || raw.trim() === "") {
    return { ok: false, error: "Not a valid number." };
  }
  if (min !== undefined && n < min) {
    return { ok: false, error: `Must be at least ${min}.` };
  }
  if (max !== undefined && n > max) {
    return { ok: false, error: `Must be at most ${max}.` };
  }
  return { ok: true, value: n };
}

export function parseAssistantBooleanValue(raw: string): boolean | null {
  const s = raw.trim().toLowerCase();
  if (s === "true" || s === "yes" || s === "1") {
    return true;
  }
  if (s === "false" || s === "no" || s === "0") {
    return false;
  }
  return null;
}

export function selectOptionsAllowedValues(options: ConfigurationSelectOption[]): Set<string> {
  const set = new Set<string>();
  for (const opt of options) {
    if (opt.value != null && opt.value !== "") {
      set.add(opt.value);
    }
  }
  return set;
}

/** Subset of field metadata + typeOptions sent to the config-assistant API (bounded JSON). */
export function buildConfigAssistantFieldContext(field: ConfigurationField): Record<string, unknown> {
  const base: Record<string, unknown> = {
    name: field.name,
    label: field.label,
    type: field.type,
    description: field.description,
    placeholder: field.placeholder,
    required: field.required,
  };
  if (field.typeOptions && Object.keys(field.typeOptions).length > 0) {
    base.typeOptions = field.typeOptions;
  }
  return base;
}
