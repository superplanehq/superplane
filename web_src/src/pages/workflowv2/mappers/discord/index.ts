import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { sendTextMessageMapper } from "./send_text_message";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendTextMessage: sendTextMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
