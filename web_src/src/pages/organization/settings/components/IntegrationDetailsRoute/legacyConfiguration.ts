import type { ConfigurationField } from "@/api-client";

const REDACTED_VALUE = "<redacted>";

export function editableConfigurationValue(field: ConfigurationField, value: unknown): unknown {
  if (field.sensitive && value === REDACTED_VALUE) {
    return undefined;
  }

  return value;
}

export function nextConfigurationValue(field: ConfigurationField, storedValue: unknown, nextValue: unknown): unknown {
  if (field.sensitive && storedValue === REDACTED_VALUE && nextValue === undefined) {
    return REDACTED_VALUE;
  }

  return nextValue;
}

export function editableConfigurationField(field: ConfigurationField, storedValue: unknown): ConfigurationField {
  if (!field.sensitive || storedValue !== REDACTED_VALUE) {
    return field;
  }

  return {
    ...field,
    placeholder: "Configured. Enter a new value to replace it.",
  };
}
