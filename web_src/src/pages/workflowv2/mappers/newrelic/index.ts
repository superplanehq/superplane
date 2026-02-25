import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIssueTriggerRenderer } from "./on_issue";
import { runNrqlQueryMapper } from "./run_nrql_query";
import { reportMetricMapper } from "./report_metric";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
    runNRQLQuery: runNrqlQueryMapper,
    reportMetric: reportMetricMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
    on_issue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
    runNRQLQuery: buildActionStateRegistry("queried"),
    reportMetric: buildActionStateRegistry("reported"),
};
