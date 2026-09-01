import type { ConfigurationField } from "@/api-client";

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
