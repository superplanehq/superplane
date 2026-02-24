import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onMentionTriggerRenderer } from "./on_mention";
import { sendMessageMapper } from "./send_message";
import { waitForButtonClickMapper, WAIT_FOR_BUTTON_CLICK_STATE_REGISTRY } from "./wait_for_button_click";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendMessage: sendMessageMapper,
  waitForButtonClick: waitForButtonClickMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onMention: onMentionTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendMessage: buildActionStateRegistry("sent"),
  waitForButtonClick: WAIT_FOR_BUTTON_CLICK_STATE_REGISTRY,
};
