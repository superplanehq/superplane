import type { ComponentBaseMapper, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIncidentMapper } from "./create_incident";
import { getIncidentMapper } from "./get_incident";
import { deleteIncidentMapper } from "./delete_incident";
import { createAlertMapper } from "./create_alert";
import { getAlertMapper } from "./get_alert";
import { deleteAlertMapper } from "./delete_alert";
import { updateAlertMapper } from "./update_alert";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIncident: createIncidentMapper,
  getIncident: getIncidentMapper,
  deleteIncident: deleteIncidentMapper,
  createAlert: createAlertMapper,
  getAlert: getAlertMapper,
  deleteAlert: deleteAlertMapper,
  updateAlert: updateAlertMapper,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIncident: buildActionStateRegistry("created"),
  getIncident: buildActionStateRegistry("fetched"),
  deleteIncident: buildActionStateRegistry("deleted"),
  createAlert: buildActionStateRegistry("created"),
  getAlert: buildActionStateRegistry("fetched"),
  deleteAlert: buildActionStateRegistry("deleted"),
  updateAlert: buildActionStateRegistry("updated"),
};
