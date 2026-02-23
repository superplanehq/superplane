import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { canvasesResolveExecutionErrors } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper, LIST_ISSUES_STATE_REGISTRY } from "./list_issues";
import { createHttpSyntheticCheckMapper } from "./create_http_synthetic_check";
import { updateHttpSyntheticCheckMapper } from "./update_http_synthetic_check";
import { deleteHttpSyntheticCheckMapper } from "./delete_http_synthetic_check";
import { buildActionStateRegistry } from "../utils";
import { onNotificationTriggerRenderer } from "./on_notification";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
  createHttpSyntheticCheck: createHttpSyntheticCheckMapper,
  updateHttpSyntheticCheck: updateHttpSyntheticCheckMapper,
  deleteHttpSyntheticCheck: deleteHttpSyntheticCheckMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onNotification: onNotificationTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listIssues: LIST_ISSUES_STATE_REGISTRY,
  queryPrometheus: buildActionStateRegistry("queried"),
  createHttpSyntheticCheck: buildActionStateRegistry("created"),
  updateHttpSyntheticCheck: buildActionStateRegistry("updated"),
  deleteHttpSyntheticCheck: buildActionStateRegistry("deleted"),
};

export async function resolveExecutionErrors(canvasId: string, executionIds: string[]) {
  return canvasesResolveExecutionErrors(
    withOrganizationHeader({
      path: { canvasId },
      body: { executionIds },
    }),
  );
}
