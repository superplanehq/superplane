import type { ComponentBaseMapper, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIncidentMapper } from "./create_incident";
import { getIncidentMapper } from "./get_incident";
import { deleteIncidentMapper } from "./delete_incident";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  getIncident: getIncidentMapper,
  deleteIncident: deleteIncidentMapper,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  getIncident: buildActionStateRegistry("fetched"),
  deleteIncident: buildActionStateRegistry("deleted"),
};
