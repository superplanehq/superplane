import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createAlertMapper } from "./create_alert";
import { deleteAlertMapper } from "./delete_alert";
import { onIssueTriggerRenderer } from "./on_issue";
import { updateAlertMapper } from "./update_alert";
import { updateIssueMapper } from "./update_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createAlert: createAlertMapper,
  deleteAlert: deleteAlertMapper,
  updateAlert: updateAlertMapper,
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createAlert: buildActionStateRegistry("created"),
  deleteAlert: buildActionStateRegistry("deleted"),
  updateAlert: buildActionStateRegistry("updated"),
  updateIssue: buildActionStateRegistry("updated"),
};
