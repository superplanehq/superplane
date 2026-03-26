import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getAlertMapper } from "./get_alert";
import { createDeployMapper } from "./create_deploy";
import { createReleaseMapper } from "./create_release";
import { getIssueMapper } from "./get_issue";
import { onIssueTriggerRenderer } from "./on_issue";
import { listAlertsMapper } from "./list_alerts";
import { updateIssueMapper } from "./update_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getAlert: getAlertMapper,
  listAlerts: listAlertsMapper,
  createDeploy: createDeployMapper,
  createRelease: createReleaseMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getAlert: buildActionStateRegistry("retrieved"),
  listAlerts: buildActionStateRegistry("listed"),
  createDeploy: buildActionStateRegistry("created"),
  createRelease: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
};
