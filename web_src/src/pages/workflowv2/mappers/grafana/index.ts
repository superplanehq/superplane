import type { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createAlertRuleMapper } from "./create_alert_rule";
import { createAnnotationMapper } from "./create_annotation";
import { createSilenceMapper } from "./create_silence";
import { deleteAlertRuleMapper } from "./delete_alert_rule";
import { deleteAnnotationMapper } from "./delete_annotation";
import { deleteSilenceMapper } from "./delete_silence";
import { getAlertRuleMapper } from "./get_alert_rule";
import { getDataSourceMapper } from "./get_data_source";
import { getSilenceMapper } from "./get_silence";
import { getSilenceEventStateRegistry } from "./get_silence_state";
import { listAlertRulesMapper } from "./list_alert_rules";
import { listAnnotationsMapper } from "./list_annotations";
import { listDataSourcesMapper } from "./list_data_sources";
import { listSilencesMapper } from "./list_silences";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { queryLogsMapper } from "./query_logs";
import { queryTracesMapper } from "./query_traces";
import { updateAlertRuleMapper } from "./update_alert_rule";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createAlertRule: createAlertRuleMapper,
  deleteAlertRule: deleteAlertRuleMapper,
  getAlertRule: getAlertRuleMapper,
  listAlertRules: listAlertRulesMapper,
  queryDataSource: queryDataSourceMapper,
  listDataSources: listDataSourcesMapper,
  getDataSource: getDataSourceMapper,
  queryLogs: queryLogsMapper,
  queryTraces: queryTracesMapper,
  updateAlertRule: updateAlertRuleMapper,
  createAnnotation: createAnnotationMapper,
  listAnnotations: listAnnotationsMapper,
  deleteAnnotation: deleteAnnotationMapper,
  createSilence: createSilenceMapper,
  deleteSilence: deleteSilenceMapper,
  getSilence: getSilenceMapper,
  listSilences: listSilencesMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createAlertRule: buildActionStateRegistry("created"),
  deleteAlertRule: buildActionStateRegistry("deleted"),
  getAlertRule: buildActionStateRegistry("fetched"),
  listAlertRules: buildActionStateRegistry("listed"),
  queryDataSource: buildActionStateRegistry("queried"),
  listDataSources: buildActionStateRegistry("listed"),
  getDataSource: buildActionStateRegistry("fetched"),
  queryLogs: buildActionStateRegistry("queried"),
  queryTraces: buildActionStateRegistry("queried"),
  updateAlertRule: buildActionStateRegistry("updated"),
  createAnnotation: buildActionStateRegistry("created"),
  listAnnotations: buildActionStateRegistry("listed"),
  deleteAnnotation: buildActionStateRegistry("deleted"),
  createSilence: buildActionStateRegistry("created"),
  deleteSilence: buildActionStateRegistry("deleted"),
  getSilence: getSilenceEventStateRegistry,
  listSilences: buildActionStateRegistry("listed"),
};
