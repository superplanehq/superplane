import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { createIncidentMapper } from "./create_incident";
import { getIncidentMapper } from "./get_incident";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  getIncident: getIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  getIncident: buildActionStateRegistry("fetched"),
};
