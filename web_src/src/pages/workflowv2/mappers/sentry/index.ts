import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createDeployMapper } from "./create_deploy";
import { createReleaseMapper } from "./create_release";
import { getAlertMapper } from "./get_alert";
import { getIssueMapper } from "./get_issue";
import { listAlertsMapper } from "./list_alerts";
import { onIssueTriggerRenderer } from "./on_issue";
import { updateIssueMapper } from "./update_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDeploy: createDeployMapper,
  createRelease: createReleaseMapper,
  getAlert: getAlertMapper,
  getIssue: getIssueMapper,
  listAlerts: listAlertsMapper,
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDeploy: buildActionStateRegistry("created"),
  createRelease: buildActionStateRegistry("created"),
  getAlert: buildActionStateRegistry("retrieved"),
  getIssue: buildActionStateRegistry("retrieved"),
  listAlerts: buildActionStateRegistry("listed"),
  updateIssue: buildActionStateRegistry("updated"),
};
