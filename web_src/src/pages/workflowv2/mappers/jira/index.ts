import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./create_issue";
import { deleteIssueMapper } from "./delete_issue";
import { getIssueMapper } from "./get_issue";
import { updateIssueMapper } from "./update_issue";
import { createIncidentMapper } from "./create_incident";
import { getIncidentMapper } from "./get_incident";
import { deleteIncidentMapper } from "./delete_incident";
import { createAlertMapper } from "./create_alert";
import { getAlertMapper } from "./get_alert";
import { deleteAlertMapper } from "./delete_alert";
import { updateAlertMapper } from "./update_alert";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
  deleteIssue: deleteIssueMapper,
  createIncident: createIncidentMapper,
  getIncident: getIncidentMapper,
  deleteIncident: deleteIncidentMapper,
  createAlert: createAlertMapper,
  getAlert: getAlertMapper,
  deleteAlert: deleteAlertMapper,
  updateAlert: updateAlertMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  deleteIssue: buildActionStateRegistry("deleted"),
  createIncident: buildActionStateRegistry("created"),
  getIncident: buildActionStateRegistry("fetched"),
  deleteIncident: buildActionStateRegistry("deleted"),
  createAlert: buildActionStateRegistry("created"),
  getAlert: buildActionStateRegistry("fetched"),
  deleteAlert: buildActionStateRegistry("deleted"),
  updateAlert: buildActionStateRegistry("updated"),
};
