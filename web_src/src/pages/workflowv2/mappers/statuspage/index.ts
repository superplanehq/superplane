import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createIncidentMapper } from "./create_incident";
import { updateIncidentMapper } from "./update_incident";
import { getIncidentMapper } from "./get_incident";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  updateIncident: updateIncidentMapper,
  getIncident: getIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  updateIncident: buildActionStateRegistry("updated"),
  getIncident: buildActionStateRegistry("fetched"),
};
