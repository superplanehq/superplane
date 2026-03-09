import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { reportMetricMapper } from "./report_metric";
import { runNRQLQueryMapper } from "./run_nrql_query";
import { onIssueTriggerRenderer, onIssueCustomFieldRenderer } from "./on_issue";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  reportMetric: reportMetricMapper,
  runNRQLQuery: runNRQLQueryMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onIssue: onIssueCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  reportMetric: buildActionStateRegistry("reported"),
  runNRQLQuery: buildActionStateRegistry("executed"),
};
