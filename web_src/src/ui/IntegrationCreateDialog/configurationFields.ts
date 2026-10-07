import type { ConfigurationField } from "@/api-client";
import { isFieldRequired, isFieldVisible } from "@/lib/components";

const WEBHOOK_SECRET_FIELD_NAMES = ["signingSecret", "webhookSigningSecret"];

/** Named fields the dialog may render, minus the ones the caller hides. */
export function selectVisibleFields(
  fields: ConfigurationField[] | undefined,
  hiddenFieldNames: string[],
): ConfigurationField[] {
  return (fields ?? []).filter((field) => Boolean(field.name) && !hiddenFieldNames.includes(field.name!));
}

/** Fields shown in the create step. Without a split, the step owns every visible field. */
export function selectCreateStepFields(
  visibleFields: ConfigurationField[],
  initialStepFieldNames: string[] | undefined,
): ConfigurationField[] {
  if (!initialStepFieldNames?.length) return visibleFields;
  return visibleFields.filter((field) => initialStepFieldNames.includes(field.name!));
}

/** Fields shown after the webhook URL: the remainder of a split, else the webhook secrets. */
export function selectWebhookStepFields(
  visibleFields: ConfigurationField[],
  initialStepFieldNames: string[] | undefined,
): ConfigurationField[] {
  if (initialStepFieldNames?.length) {
    return visibleFields.filter((field) => !initialStepFieldNames.includes(field.name!));
  }
  return visibleFields.filter((field) => WEBHOOK_SECRET_FIELD_NAMES.includes(field.name!));
}

/** True when every visible, required create-step field has a non-empty value. */
export function areRequiredCreateFieldsFilled(fields: ConfigurationField[], values: Record<string, unknown>): boolean {
  return fields.every((field) => isCreateFieldReady(field, values));
}

function isCreateFieldReady(field: ConfigurationField, values: Record<string, unknown>): boolean {
  if (!field.name || !isFieldVisible(field, values)) {
    return true;
  }

  const value = values[field.name];
  if (field.togglable && (value === null || value === undefined)) {
    return true;
  }
  if (!isFieldRequired(field, values)) {
    return true;
  }

  return !isConfigurationValueEmpty(value);
}

function isConfigurationValueEmpty(value: unknown): boolean {
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
