import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { GET_LAST_MENTION_STATE_REGISTRY, getLastMentionMapper } from "./get_last_mention";
import { sendTextMessageMapper } from "./send_text_message";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
  getLastMention: getLastMentionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getLastMention: GET_LAST_MENTION_STATE_REGISTRY,
};
