import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { canvasesResolveExecutionErrors } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper, LIST_ISSUES_STATE_REGISTRY } from "./list_issues";
import { buildActionStateRegistry } from "../utils";
import { sendLogEventMapper } from "./send_log_event";
import { getCheckDetailsMapper } from "./get_check_details";
import { createSyntheticCheckMapper } from "./create_synthetic_check";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
  sendLogEvent: sendLogEventMapper,
  getCheckDetails: getCheckDetailsMapper,
  createSyntheticCheck: createSyntheticCheckMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listIssues: LIST_ISSUES_STATE_REGISTRY,
  queryPrometheus: buildActionStateRegistry("queried"),
  sendLogEvent: buildActionStateRegistry("sent"),
  getCheckDetails: buildActionStateRegistry("retrieved"),
  createSyntheticCheck: buildActionStateRegistry("created"),
};

export async function resolveExecutionErrors(canvasId: string, executionIds: string[]) {
  return canvasesResolveExecutionErrors(
    withOrganizationHeader({
      path: { canvasId },
      body: { executionIds },
    }),
  );
}
