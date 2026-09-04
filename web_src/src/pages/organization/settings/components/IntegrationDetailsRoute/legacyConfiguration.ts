import type { ConfigurationField } from "@/api-client";

const REDACTED_VALUE = "<redacted>";

export function editableConfigurationValue(field: ConfigurationField, value: unknown): unknown {
  if (field.sensitive && value === REDACTED_VALUE) {
    return field.togglable ? "" : undefined;
  }

  return value;
}

export function nextConfigurationValue(field: ConfigurationField, storedValue: unknown, nextValue: unknown): unknown {
  if (!field.sensitive) {
    return nextValue;
  }

  // The field renderer uses null to mean "toggle off". Empty string means
  // "toggle on with no value yet". Keep null so the switch can turn off.
  if (nextValue === null) {
    return null;
  }

  if (storedValue === REDACTED_VALUE && (nextValue === undefined || nextValue === "")) {
    return REDACTED_VALUE;
  }

  return nextValue;
}

export function configurationSubmitPayload(
  fields: ConfigurationField[] | undefined,
  configuration: Record<string, unknown>,
): Record<string, unknown> {
  if (!fields?.length) {
    return configuration;
  }

  const payload = { ...configuration };
  for (const field of fields) {
    if (!field.name || !field.sensitive) {
      continue;
    }

    if (payload[field.name] === null) {
      payload[field.name] = "";
    }
  }

  return payload;
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
