import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { getAlertMapper } from "./get_alert";
import { onAlertTriggerRenderer } from "./on_alert";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getAlert: getAlertMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlert: onAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getAlert: buildActionStateRegistry("retrieved"),
};
