import type { ComponentsIntegrationRef, ConfigurationField } from "@/api-client";
import { isFieldRequired, validateFieldForSubmission } from "@/lib/components";

export function buildAutosaveSnapshot(
  configuration: Record<string, unknown>,
  nodeName: string,
  integrationRef?: ComponentsIntegrationRef,
): string {
  return JSON.stringify({
    configuration,
    nodeName,
    integrationRef: integrationRef
      ? {
          id: integrationRef.id || "",
          name: integrationRef.name || "",
        }
      : null,
  });
}

export function isFieldEmpty(value: unknown): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return false;
}

export function validateNestedFields(
  fields: ConfigurationField[],
  values: Record<string, unknown>,
  parentPath: string = "",
): Set<string> {
  const errors = new Set<string>();

  fields.forEach((field) => {
    if (!field.name) return;

    const fieldPath = parentPath ? `${parentPath}.${field.name}` : field.name;
    const value = values[field.name];

    const fieldIsRequired = field.required || isFieldRequired(field, values);
    if (fieldIsRequired && isFieldEmpty(value)) {
      errors.add(fieldPath);
    }

    if (value !== undefined && value !== null && value !== "") {
      const validationErrors = validateFieldForSubmission(field, value);

      if (validationErrors.length > 0) {
        errors.add(fieldPath);
      }
    }

    if (field.type === "list" && Array.isArray(value) && field.typeOptions?.list?.itemDefinition) {
      const itemSchema = field.typeOptions.list.itemDefinition.schema;
      if (itemSchema) {
        value.forEach((item, index) => {
          if (typeof item === "object" && item !== null) {
            const nestedErrors = validateNestedFields(
              itemSchema,
              item as Record<string, unknown>,
              `${fieldPath}[${index}]`,
            );
            nestedErrors.forEach((error) => errors.add(error));
          }
        });
      }
    } else if (
      field.type === "object" &&
      typeof value === "object" &&
      value !== null &&
      field.typeOptions?.object?.schema
    ) {
      const nestedErrors = validateNestedFields(
        field.typeOptions.object.schema,
        value as Record<string, unknown>,
        fieldPath,
      );
      nestedErrors.forEach((error) => errors.add(error));
    }
  });

  return errors;
}

export function shouldAutosaveOnChangeByFieldType(fieldType: ConfigurationField["type"] | undefined): boolean {
  if (!fieldType) {
    return false;
  }

  return ![
    "string",
    "text",
    "xml",
    "expression",
    "number",
    "url",
    "date",
    "datetime",
    "time",
    "cron",
    "git-ref",
    "app",
  ].includes(fieldType);
}
