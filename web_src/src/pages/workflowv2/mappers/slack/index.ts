import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onAppMentionTriggerRenderer } from "./on_app_mention";
import { sendTextMessageMapper } from "./send_text_message";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAppMention: onAppMentionTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
