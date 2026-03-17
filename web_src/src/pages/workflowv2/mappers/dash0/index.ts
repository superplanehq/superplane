import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { canvasesResolveExecutionErrors } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper, LIST_ISSUES_STATE_REGISTRY } from "./list_issues";
import { createHttpSyntheticCheckMapper } from "./create_http_synthetic_check";
import { updateHttpSyntheticCheckMapper } from "./update_http_synthetic_check";
import { deleteHttpSyntheticCheckMapper } from "./delete_http_synthetic_check";
import { getHttpSyntheticCheckMapper, GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY } from "./get_http_synthetic_check";
import { createCheckRuleMapper } from "./create_check_rule";
import { getCheckRuleMapper } from "./get_check_rule";
import { updateCheckRuleMapper } from "./update_check_rule";
import { deleteCheckRuleMapper } from "./delete_check_rule";
import { sendLogEventMapper } from "./send_log_event";
import { buildActionStateRegistry } from "../utils";
import { onAlertNotificationTriggerRenderer } from "./on_alert_notification";
import { onSyntheticCheckNotificationTriggerRenderer } from "./on_synthetic_check_notification";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
  createHttpSyntheticCheck: createHttpSyntheticCheckMapper,
  updateHttpSyntheticCheck: updateHttpSyntheticCheckMapper,
  deleteHttpSyntheticCheck: deleteHttpSyntheticCheckMapper,
  getHttpSyntheticCheck: getHttpSyntheticCheckMapper,
  createCheckRule: createCheckRuleMapper,
  getCheckRule: getCheckRuleMapper,
  updateCheckRule: updateCheckRuleMapper,
  deleteCheckRule: deleteCheckRuleMapper,
  sendLogEvent: sendLogEventMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertNotification: onAlertNotificationTriggerRenderer,
  onSyntheticCheckNotification: onSyntheticCheckNotificationTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listIssues: LIST_ISSUES_STATE_REGISTRY,
  queryPrometheus: buildActionStateRegistry("queried"),
  createHttpSyntheticCheck: buildActionStateRegistry("created"),
  updateHttpSyntheticCheck: buildActionStateRegistry("updated"),
  deleteHttpSyntheticCheck: buildActionStateRegistry("deleted"),
  getHttpSyntheticCheck: GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY,
  createCheckRule: buildActionStateRegistry("created"),
  getCheckRule: buildActionStateRegistry("fetched"),
  updateCheckRule: buildActionStateRegistry("updated"),
  deleteCheckRule: buildActionStateRegistry("deleted"),
  sendLogEvent: buildActionStateRegistry("sent"),
};

export async function resolveExecutionErrors(canvasId: string, executionIds: string[]) {
  return canvasesResolveExecutionErrors(
    withOrganizationHeader({
      path: { canvasId },
      body: { executionIds },
    }),
  );
}
