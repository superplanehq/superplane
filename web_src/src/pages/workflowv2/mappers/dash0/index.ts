import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { canvasesResolveExecutionErrors } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper, LIST_ISSUES_STATE_REGISTRY } from "./list_issues";
import { buildActionStateRegistry } from "../utils";
import { onAlertEventTriggerRenderer } from "./on_alert_event";
import { sendLogEventMapper } from "./send_log_event";
import { getCheckDetailsMapper } from "./get_check_details";
import { createSyntheticCheckMapper } from "./create_synthetic_check";
import { updateSyntheticCheckMapper } from "./update_synthetic_check";
import { createCheckRuleMapper } from "./create_check_rule";
import { updateCheckRuleMapper } from "./update_check_rule";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
  sendLogEvent: sendLogEventMapper,
  getCheckDetails: getCheckDetailsMapper,
  createSyntheticCheck: createSyntheticCheckMapper,
  updateSyntheticCheck: updateSyntheticCheckMapper,
  createCheckRule: createCheckRuleMapper,
  updateCheckRule: updateCheckRuleMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertEvent: onAlertEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listIssues: LIST_ISSUES_STATE_REGISTRY,
  queryPrometheus: buildActionStateRegistry("queried"),
  sendLogEvent: buildActionStateRegistry("sent"),
  getCheckDetails: buildActionStateRegistry("retrieved"),
  createSyntheticCheck: buildActionStateRegistry("created"),
  updateSyntheticCheck: buildActionStateRegistry("updated"),
  createCheckRule: buildActionStateRegistry("created"),
  updateCheckRule: buildActionStateRegistry("updated"),
};

export async function resolveExecutionErrors(canvasId: string, executionIds: string[]) {
  return canvasesResolveExecutionErrors(
    withOrganizationHeader({
      path: { canvasId },
      body: { executionIds },
    }),
  );
}
