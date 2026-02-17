import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIncidentTriggerRenderer } from "./on_incident";
import { createIncidentMapper } from "./create_incident";
import { createEventMapper } from "./create_event";
import { updateIncidentMapper } from "./update_incident";
import { getIncidentMapper } from "./get_incident";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  createEvent: createEventMapper,
  updateIncident: updateIncidentMapper,
  getIncident: getIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIncident: onIncidentTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  createEvent: buildActionStateRegistry("created"),
  updateIncident: buildActionStateRegistry("updated"),
  getIncident: buildActionStateRegistry("retrieved"),
};
