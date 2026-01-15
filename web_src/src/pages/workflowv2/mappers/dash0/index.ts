import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { queryPrometheusMapper } from "./query_prometheus";
import { onIssueStatusTriggerRenderer } from "./on_issue_status";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssueStatus: onIssueStatusTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
