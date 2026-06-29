import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createAlertMapper } from "./create_alert";
import { createDeployMapper } from "./create_deploy";
import { createReleaseMapper } from "./create_release";
import { deleteAlertMapper } from "./delete_alert";
import { getAlertMapper } from "./get_alert";
import { getIssueMapper } from "./get_issue";
import { listAlertsMapper } from "./list_alerts";
import { onIssueTriggerRenderer } from "./on_issue";
import { updateAlertMapper } from "./update_alert";
import { updateIssueMapper } from "./update_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createAlert: createAlertMapper,
  createDeploy: createDeployMapper,
  createRelease: createReleaseMapper,
  deleteAlert: deleteAlertMapper,
  getAlert: getAlertMapper,
  getIssue: getIssueMapper,
  listAlerts: listAlertsMapper,
  updateAlert: updateAlertMapper,
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createAlert: buildActionStateRegistry("created"),
  createDeploy: buildActionStateRegistry("created"),
  createRelease: buildActionStateRegistry("created"),
  deleteAlert: buildActionStateRegistry("deleted"),
  getAlert: buildActionStateRegistry("retrieved"),
  getIssue: buildActionStateRegistry("retrieved"),
  listAlerts: buildActionStateRegistry("listed"),
  updateAlert: buildActionStateRegistry("updated"),
  updateIssue: buildActionStateRegistry("updated"),
};
