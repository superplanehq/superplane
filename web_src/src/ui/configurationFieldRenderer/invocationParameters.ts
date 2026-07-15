import type { ConfigurationField } from "@/api-client";

export function normalizeInvocationParameterDefinitions(raw: unknown): ConfigurationField[] {
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
      defaultValue: param.default ?? param.defaultValue,
      typeOptions: param.typeOptions as ConfigurationField["typeOptions"],
    });
  }

  return fields;
}
