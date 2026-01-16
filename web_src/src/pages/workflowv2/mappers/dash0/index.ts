import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { queryPrometheusMapper } from "./query_prometheus";
import { listIssuesMapper } from "./list_issues";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
  listIssues: listIssuesMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
