import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";

import { createEventMapper } from "./create_event";
import { onAlertFiredTriggerRenderer } from "./on_alert_fired";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createEvent: createEventMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFired: onAlertFiredTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createEvent: buildActionStateRegistry("Sent"),
};
