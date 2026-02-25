import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onMentionTriggerRenderer } from "./on_mention";
import { sendMessageMapper } from "./send_message";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendMessage: sendMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onMention: onMentionTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendMessage: buildActionStateRegistry("sent"),
};
