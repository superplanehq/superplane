import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIncidentTriggerRenderer } from "./on_incident";
import { onIncidentStatusUpdateTriggerRenderer } from "./on_incident_status_update";
import { createIncidentMapper } from "./create_incident";
import { updateIncidentMapper } from "./update_incident";
import { listIncidentsMapper, LIST_INCIDENTS_STATE_REGISTRY } from "./list_incidents";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  updateIncident: updateIncidentMapper,
  listIncidents: listIncidentsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIncident: onIncidentTriggerRenderer,
  onIncidentStatusUpdate: onIncidentStatusUpdateTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  updateIncident: buildActionStateRegistry("updated"),
  listIncidents: LIST_INCIDENTS_STATE_REGISTRY,
};
