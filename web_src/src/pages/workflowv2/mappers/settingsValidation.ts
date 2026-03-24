import type { ConfigurationField } from "@/api-client";
import { discordSendTextMessageSettingsValidation } from "./discord/send_text_message";

export interface RealtimeValidationError {
  field: string;
  message: string;
  type: "validation_rule" | "required" | "visibility";
}

export interface SettingsValidationContext {
  blockName?: string;
  integrationName?: string;
  configurationFields: ConfigurationField[];
  values: Record<string, unknown>;
}

type SettingsValidationFn = (context: SettingsValidationContext) => RealtimeValidationError[];

const SETTINGS_VALIDATORS: Record<string, SettingsValidationFn> = {
  "discord.sendTextMessage": discordSendTextMessageSettingsValidation,
};

export function getSettingsRealtimeValidationErrors(context: SettingsValidationContext): RealtimeValidationError[] {
  const blockName = context.blockName;
  if (!blockName) return [];

  const validator = SETTINGS_VALIDATORS[blockName];
  if (!validator) return [];

  return validator(context);
}
