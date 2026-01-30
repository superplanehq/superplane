import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createEventMapper } from "./create_event";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createEvent: createEventMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createEvent: buildActionStateRegistry("Event created"),
};
