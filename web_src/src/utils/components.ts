/**
 * Functions for rendering forms with proper labels and placeholders.
 */

import type { ConfigurationField } from "../api-client";

/**
 * Validates a cron expression
 */
function validateCronExpression(cronExpression: string): string | null {
  if (!cronExpression || cronExpression.trim() === "") {
    return "Cron expression cannot be empty";
  }

  const trimmed = cronExpression.trim();

  // Quick check for obviously invalid expressions to avoid expensive parsing
  if (trimmed.length < 5) {
    return "Cron expression too short";
  }

  const parts = trimmed.split(/\s+/);

  // Cron expressions should have 6 parts (second minute hour day month dayofweek)
  if (parts.length !== 6 && parts.length !== 5) {
    return `Expected 5 or 6 fields, got ${parts.length}`;
  }

  // Quick validation with pre-compiled regex for better performance
  const validChars = /^[0-9*,\-/A-Z]+$/;

  // Use a more efficient approach - validate only the problematic parts
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];

    if (!validChars.test(part)) {
      return "Invalid characters. Use only: numbers, *, ,, -, / and day names";
    }

    // Skip expensive range checks for wildcards and complex expressions
    if (part === "*" || part.includes(",") || part.includes("-") || part.includes("/")) {
      continue;
    }

    const num = parseInt(part);
    if (!isNaN(num)) {
      // Basic range validation with early exit
      switch (i) {
        case 0: // second (for 6-field format)
        case 1: // minute
          if (num < 0 || num > 59) return "Invalid minute/second value";
          break;
        case 2: // hour
          if (num < 0 || num > 23) return "Invalid hour value";
          break;
        case 3: // day
          if (num < 1 || num > 31) return "Invalid day value";
          break;
        case 4: // month
          if (num < 1 || num > 12) return "Invalid month value";
          break;
        case 5: // dayofweek
          if (num < 0 || num > 6) return "Invalid day of week value";
          break;
      }
    }
  }

  return null;
}

export function getDefaultEventType(sourceType: string): string {
  switch (sourceType) {
    case "github":
      return "push";
    case "semaphore":
      return "pipeline_done";
    default:
      return "";
  }
}

export function getDefaultFilterExpression(sourceType: string): string {
  switch (sourceType) {
    case "github":
      return '$.ref == "refs/heads/main"';
    case "semaphore":
      return '$.pipeline.state == "done"';
    default:
      return "";
  }
}

export function getResourceType(sourceType: string): string {
  switch (sourceType) {
    case "github":
      return "repository";
    case "semaphore":
      return "project";
    default:
      return "";
  }
}

export function getResourceLabel(sourceType: string): string {
  const type = getResourceType(sourceType);
  return type.charAt(0).toUpperCase() + type.slice(1);
}

export function getResourcePlaceholder(sourceType: string): string {
  switch (sourceType) {
    case "github":
      return "my-repository";
    case "semaphore":
      return "my-semaphore-project";
    default:
      return "";
  }
}

export function getIntegrationLabel(sourceType: string): string {
  switch (sourceType) {
    case "github":
      return "GitHub repository";
    case "semaphore":
      return "Semaphore project";
    default:
      return "";
  }
}

export function getEventTypePlaceholder(sourceType: string): string {
  switch (sourceType) {
    case "github":
      return "e.g., push, pull_request, deployment";
    case "semaphore":
      return "e.g., pipeline_done";
    default:
      return "Event type";
  }
}

/*
 * Returns true if the source type is an event source that doesn't require an integration
 * (manual, scheduled, webhook).
 */
export function isRegularEventSource(sourceType: string): boolean {
  return sourceType === "manual" || sourceType === "scheduled" || sourceType === "webhook";
}

/**
 * Checks if a field is visible based on its visibility conditions
 */
export function isFieldVisible(field: ConfigurationField, allValues: Record<string, unknown>): boolean {
  if (!field.visibilityConditions || field.visibilityConditions.length === 0) {
    return true;
  }

  // All conditions must be satisfied (AND logic)
  return field.visibilityConditions.every((condition) => {
    if (!condition.field || !condition.values) {
      return true;
    }

    const fieldValue = allValues[condition.field];

    // Convert field value to string for comparison
    const fieldValueStr = fieldValue !== undefined && fieldValue !== null ? String(fieldValue) : "";

    // Check if the field value matches any of the expected values
    // Support wildcard "*" to match any non-empty value
    return condition.values.some((expectedValue) => {
      if (expectedValue === "*") {
        // Wildcard matches any non-empty value
        return fieldValueStr !== "";
      }
      return fieldValueStr === expectedValue;
    });
  });
}

/**
 * Filters a configuration object to only include visible fields based on their visibility conditions.
 * This ensures that hidden fields are not included in the API payload.
 *
 * @param configuration - The full configuration object with all field values
 * @param fields - The configuration field definitions including visibility conditions
 * @returns A filtered configuration object containing only visible fields
 */
export function filterVisibleConfiguration(
  configuration: Record<string, unknown>,
  fields: ConfigurationField[],
): Record<string, unknown> {
  const filtered: Record<string, unknown> = {};

  for (const field of fields) {
    if (!field.name || !isFieldVisible(field, configuration)) {
      continue;
    }

    const fieldValue = configuration[field.name];
    if (fieldValue === undefined) {
      continue;
    }

    filtered[field.name] = filterFieldValueByVisibility(field, fieldValue);
  }

  return filtered;
}

function filterFieldValueByVisibility(field: ConfigurationField, value: unknown): unknown {
  const objectSchema = field.typeOptions?.object?.schema;
  if (field.type === "object" && objectSchema && isRecord(value)) {
    return filterVisibleConfiguration(value, objectSchema);
  }

  const listItemSchema = field.typeOptions?.list?.itemDefinition?.schema;
  if (field.type === "list" && listItemSchema && Array.isArray(value)) {
    return value.map((item) => (isRecord(item) ? filterVisibleConfiguration(item, listItemSchema) : item));
  }

  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * Checks if a field is required based on its required conditions
 */
export function isFieldRequired(field: ConfigurationField, allValues: Record<string, unknown>): boolean {
  // If field is always required, return true
  if (field.required) {
    return true;
  }

  // If there are no required conditions, field is not required
  if (!field.requiredConditions || field.requiredConditions.length === 0) {
    return false;
  }

  // Check if any required condition is satisfied (OR logic)
  return field.requiredConditions.some((condition) => {
    if (!condition.field || !condition.values) {
      return false;
    }

    const fieldValue = allValues[condition.field];
    const fieldValueStr = fieldValue !== undefined && fieldValue !== null ? String(fieldValue) : "";

    // Check if the field value matches any of the expected values
    return condition.values.some((expectedValue) => {
      return fieldValueStr === expectedValue;
    });
  });
}

/**
 * Validates a single field value for form submission (includes type-specific validation)
 */
export function validateFieldForSubmission(field: ConfigurationField, value: unknown): string[] {
  const errors: string[] = [];

  // Add type-specific validation for form submission
  if (field.type === "cron" && value != null && value !== "") {
    const cronError = validateCronExpression(String(value));
    if (cronError) {
      errors.push(cronError);
    }
  }

  // Add min/max validation for number fields
  if (field.type === "number" && value != null && value !== "") {
    const numValue = Number(value);
    if (!isNaN(numValue) && field.typeOptions?.number) {
      const { min, max } = field.typeOptions.number;
      if (min !== undefined && numValue < min) {
        errors.push(`Value must be at least ${min}`);
      }
      if (max !== undefined && numValue > max) {
        errors.push(`Value must not exceed ${max}`);
      }
    }
  }

  return errors;
}

/**
 * Parses default values based on field type to match API expectations
 */
export function parseDefaultValues(configurationFields: ConfigurationField[]): Record<string, unknown> {
  return configurationFields
    .map((field) => [field.name, field.defaultValue, field.type] as const)
    .reduce(
      (acc, [name, defaultValue, fieldType]) => {
        if (name && defaultValue != null) {
          // Parse defaultValue based on field type
          let parsedValue: unknown = defaultValue;

          if (typeof defaultValue === "string" && defaultValue !== "") {
            switch (fieldType) {
              case "number": {
                const num = Number(defaultValue);
                if (!isNaN(num)) {
                  parsedValue = num;
                }
                break;
              }
              case "boolean": {
                parsedValue = defaultValue === "true";
                break;
              }
              case "multi-select":
              case "days-of-week":
              case "list":
              case "any-predicate-list": {
                try {
                  parsedValue = JSON.parse(defaultValue);
                } catch {
                  // If parsing fails, treat as single item array for multi-select
                  if (fieldType === "multi-select") {
                    parsedValue = [defaultValue];
                  }
                }
                break;
              }
              case "object": {
                try {
                  parsedValue = JSON.parse(defaultValue);
                } catch {
                  // If parsing fails, keep as empty object
                  parsedValue = {};
                }
                break;
              }
              case "timezone": {
                if (defaultValue === "current") {
                  const offset = -new Date().getTimezoneOffset() / 60;
                  parsedValue = offset.toString();
                } else {
                  parsedValue = defaultValue;
                }
                break;
              }
              // For string, select, date, time, datetime, day-in-year, cron, url, integration, etc.
              // keep as string
              default:
                parsedValue = defaultValue;
                break;
            }
          }

          acc[name] = parsedValue;
        }
        return acc;
      },
      {} as Record<string, unknown>,
    );
}
