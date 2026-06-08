import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./create_issue";
import { deleteIssueMapper } from "./delete_issue";
import { getIssueMapper } from "./get_issue";
import { updateIssueMapper } from "./update_issue";
import { createIncidentMapper } from "./create_incident";
import { getIncidentMapper } from "./get_incident";
import { deleteIncidentMapper } from "./delete_incident";
import { onIssueTriggerRenderer } from "./on_issue";
import { createHeartbeatMapper } from "./create_heartbeat";
import { pingHeartbeatMapper } from "./ping_heartbeat";
import { updateHeartbeatMapper } from "./update_heartbeat";
import { deleteHeartbeatMapper } from "./delete_heartbeat";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
  deleteIssue: deleteIssueMapper,
  createIncident: createIncidentMapper,
  getIncident: getIncidentMapper,
  deleteIncident: deleteIncidentMapper,
  createHeartbeat: createHeartbeatMapper,
  pingHeartbeat: pingHeartbeatMapper,
  updateHeartbeat: updateHeartbeatMapper,
  deleteHeartbeat: deleteHeartbeatMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  deleteIssue: buildActionStateRegistry("deleted"),
  createIncident: buildActionStateRegistry("created"),
  getIncident: buildActionStateRegistry("fetched"),
  deleteIncident: buildActionStateRegistry("deleted"),
  createHeartbeat: buildActionStateRegistry("created"),
  pingHeartbeat: buildActionStateRegistry("pinged"),
  updateHeartbeat: buildActionStateRegistry("updated"),
  deleteHeartbeat: buildActionStateRegistry("deleted"),
};
