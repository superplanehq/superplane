import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { onIncidentTriggerRenderer, onIncidentCustomFieldRenderer } from "./on_incident";
import { createIncidentMapper } from "./create_incident";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIncident: onIncidentTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onIncident: onIncidentCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
};
