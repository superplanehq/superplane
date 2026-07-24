import type { ConfigurationField } from "@/api-client";

export function coerceRunParameterValues(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }

  return {};
}

export function getSyncedRunParameterValues(
  parameterValues: Record<string, unknown>,
  parameterDefinitions: ConfigurationField[],
): Record<string, unknown> | null {
  if (parameterDefinitions.length === 0) {
    return Object.keys(parameterValues).length > 0 ? {} : null;
  }

  const definedNames = new Set(
    parameterDefinitions.map((definition) => definition.name).filter((name): name is string => Boolean(name)),
  );
  const hasStaleKeys = Object.keys(parameterValues).some((name) => !definedNames.has(name));
  if (!hasStaleKeys) {
    return null;
  }

  return Object.fromEntries(Object.entries(parameterValues).filter(([name]) => definedNames.has(name)));
}

export function normalizeRunParameterDefinitions(raw: unknown): ConfigurationField[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [];
  }

  const fields: ConfigurationField[] = [];

  for (const item of raw) {
    if (!item || typeof item !== "object" || Array.isArray(item)) {
      continue;
    }

    const param = item as Record<string, unknown>;
    const name = typeof param.name === "string" ? param.name.trim() : "";
    if (!name) {
      continue;
    }

    const type = typeof param.type === "string" && param.type.length > 0 ? param.type : "string";
    const label = typeof param.label === "string" && param.label.trim().length > 0 ? param.label.trim() : name;

    fields.push({
      name,
      label,
      type,
      description: typeof param.description === "string" ? param.description : undefined,
      required: param.required === true,
      defaultValue: (param.default ?? param.defaultValue) as ConfigurationField["defaultValue"],
      typeOptions: param.typeOptions as ConfigurationField["typeOptions"],
    });
  }

  return fields;
}
