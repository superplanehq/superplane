import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createIncidentMapper } from "./create_incident";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
};
