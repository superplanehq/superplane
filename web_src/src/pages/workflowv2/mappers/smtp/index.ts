import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { sendEmailMapper } from "./send_email";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  sendEmail: sendEmailMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  sendEmail: buildActionStateRegistry("sent"),
};
