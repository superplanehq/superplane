import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getAlertMapper } from "./get_alert";
import { onIssueTriggerRenderer } from "./on_issue";
import { listAlertsMapper } from "./list_alerts";
import { updateIssueMapper } from "./update_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getAlert: getAlertMapper,
  listAlerts: listAlertsMapper,
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getAlert: buildActionStateRegistry("retrieved"),
  listAlerts: buildActionStateRegistry("listed"),
  updateIssue: buildActionStateRegistry("updated"),
};
