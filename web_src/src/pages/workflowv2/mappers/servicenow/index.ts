import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { createIncidentMapper } from "./create_incident";
import { getIncidentsMapper, GET_INCIDENTS_STATE_REGISTRY } from "./get_incidents";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  getIncidents: getIncidentsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  getIncidents: GET_INCIDENTS_STATE_REGISTRY,
};
