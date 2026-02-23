import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { getAlertMapper } from "./get_alert";
import { createSilenceMapper } from "./create_silence";
import { expireSilenceMapper } from "./expire_silence";
import { onAlertCustomFieldRenderer, onAlertTriggerRenderer } from "./on_alert";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getAlert: getAlertMapper,
  createSilence: createSilenceMapper,
  expireSilence: expireSilenceMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlert: onAlertTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onAlert: onAlertCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getAlert: buildActionStateRegistry("retrieved"),
  createSilence: buildActionStateRegistry("created"),
  expireSilence: buildActionStateRegistry("expired"),
};
