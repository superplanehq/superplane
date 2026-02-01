import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIncidentCreatedTriggerRenderer } from "./on_incident_created";
import { onIncidentResolvedTriggerRenderer } from "./on_incident_resolved";
import { createIncidentMapper } from "./create_incident";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIncidentCreated: onIncidentCreatedTriggerRenderer,
  onIncidentResolved: onIncidentResolvedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
};
