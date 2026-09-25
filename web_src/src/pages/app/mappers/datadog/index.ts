import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createEventMapper } from "./create_event";
import { onErrorTrackingAlertTriggerRenderer } from "./on_error_tracking_alert";
import { buildActionStateRegistry } from "../eventDisplay";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createEvent: createEventMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onErrorTrackingAlert: onErrorTrackingAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createEvent: buildActionStateRegistry("Event created"),
  onErrorTrackingAlert: buildActionStateRegistry("triggered"),
};
