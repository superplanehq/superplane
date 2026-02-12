import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { canvasesResolveExecutionErrors } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper, LIST_ISSUES_STATE_REGISTRY } from "./list_issues";
import { buildActionStateRegistry } from "../utils";
import { createCheckRuleMapper } from "./create_check_rule";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
  createCheckRule: createCheckRuleMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listIssues: LIST_ISSUES_STATE_REGISTRY,
  queryPrometheus: buildActionStateRegistry("queried"),
  createCheckRule: buildActionStateRegistry("created"),
};

export async function resolveExecutionErrors(canvasId: string, executionIds: string[]) {
  return canvasesResolveExecutionErrors(
    withOrganizationHeader({
      path: { canvasId },
      body: { executionIds },
    }),
  );
}
