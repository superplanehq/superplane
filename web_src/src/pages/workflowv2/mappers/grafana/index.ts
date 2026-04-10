import type { ComponentBaseMapper, CustomFieldRenderer, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { listDataSourcesMapper } from "./list_data_sources";
import { getDataSourceMapper } from "./get_data_source";
import { queryLogsMapper } from "./query_logs";
import { queryTracesMapper } from "./query_traces";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryDataSource: queryDataSourceMapper,
  listDataSources: listDataSourcesMapper,
  getDataSource: getDataSourceMapper,
  queryLogs: queryLogsMapper,
  queryTraces: queryTracesMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  queryDataSource: buildActionStateRegistry("queried"),
  listDataSources: buildActionStateRegistry("listed"),
  getDataSource: buildActionStateRegistry("fetched"),
  queryLogs: buildActionStateRegistry("queried"),
  queryTraces: buildActionStateRegistry("queried"),
};
