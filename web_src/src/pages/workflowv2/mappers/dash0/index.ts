import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { workflowsResolveExecutionErrors } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper, LIST_ISSUES_STATE_REGISTRY } from "./list_issues";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listIssues: LIST_ISSUES_STATE_REGISTRY,
  queryPrometheus: buildActionStateRegistry("queried"),
};

export async function resolveExecutionErrors(workflowId: string, executionIds: string[]) {
  return workflowsResolveExecutionErrors(
    withOrganizationHeader({
      path: { workflowId },
      body: { executionIds },
    }),
  );
}
