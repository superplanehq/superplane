import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { queryPrometheusMapper } from "./query_prometheus";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryPrometheus: queryPrometheusMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
