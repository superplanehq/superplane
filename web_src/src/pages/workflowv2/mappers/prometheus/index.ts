import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { getAlertMapper } from "./get_alert";
import { onAlertCustomFieldRenderer, onAlertTriggerRenderer } from "./on_alert";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getAlert: getAlertMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlert: onAlertTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onAlert: onAlertCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getAlert: buildActionStateRegistry("retrieved"),
};
