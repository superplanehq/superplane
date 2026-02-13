import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryDataSource: queryDataSourceMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  queryDataSource: buildActionStateRegistry("queried"),
};
