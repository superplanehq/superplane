import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onMonitorAlertTriggerRenderer } from "./on_monitor_alert";
import { createEventMapper } from "./create_event";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createEvent: createEventMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onMonitorAlert: onMonitorAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createEvent: buildActionStateRegistry("created"),
};
