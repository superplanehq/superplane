import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onAppMentionTriggerRenderer } from "./on_app_mention";
import { sendTextMessageMapper } from "./send_text_message";
import { waitForButtonClickMapper, WAIT_FOR_BUTTON_CLICK_STATE_REGISTRY } from "./wait_for_button_click";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
  waitForButtonClick: waitForButtonClickMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAppMention: onAppMentionTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendTextMessage: buildActionStateRegistry("sent"),
  waitForButtonClick: WAIT_FOR_BUTTON_CLICK_STATE_REGISTRY,
};
