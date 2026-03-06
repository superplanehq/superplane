import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onMentionTriggerRenderer } from "./on_mention";
import { onMessageTriggerRenderer } from "./on_message";
import { sendTextMessageMapper } from "./send_text_message";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onMention: onMentionTriggerRenderer,
  onMessage: onMessageTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendTextMessage: buildActionStateRegistry("sent"),
};
