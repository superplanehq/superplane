import type { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createAlertRuleMapper } from "./create_alert_rule";
import { createAnnotationMapper } from "./create_annotation";
import { createHttpSyntheticCheckMapper } from "./create_http_synthetic_check";
import { createSilenceMapper } from "./create_silence";
import { deleteAlertRuleMapper } from "./delete_alert_rule";
import { deleteAnnotationMapper } from "./delete_annotation";
import { deleteHttpSyntheticCheckMapper } from "./delete_http_synthetic_check";
import { deleteSilenceMapper } from "./delete_silence";
import { addIncidentActivityMapper } from "./add_incident_activity";
import { declareDrillMapper, declareIncidentMapper } from "./declare_incident";
import { getAlertRuleMapper } from "./get_alert_rule";
import { getDashboardMapper } from "./get_dashboard";
import { getIncidentMapper } from "./get_incident";
import { getSilenceMapper } from "./get_silence";
import { getSilenceEventStateRegistry } from "./get_silence_state";
import { getHttpSyntheticCheckMapper, GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY } from "./get_http_synthetic_check";
import { listAlertRulesMapper } from "./list_alert_rules";
import { listAnnotationsMapper } from "./list_annotations";
import { listSilencesMapper } from "./list_silences";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { queryLogsMapper } from "./query_logs";
import { queryTracesMapper } from "./query_traces";
import { renderPanelMapper } from "./render_panel";
import { resolveIncidentMapper } from "./resolve_incident";
import { updateHttpSyntheticCheckMapper } from "./update_http_synthetic_check";
import { updateAlertRuleMapper } from "./update_alert_rule";
import { updateIncidentMapper } from "./update_incident";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createAlertRule: createAlertRuleMapper,
  createHttpSyntheticCheck: createHttpSyntheticCheckMapper,
  deleteAlertRule: deleteAlertRuleMapper,
  getAlertRule: getAlertRuleMapper,
  getDashboard: getDashboardMapper,
  getHttpSyntheticCheck: getHttpSyntheticCheckMapper,
  listAlertRules: listAlertRulesMapper,
  queryDataSource: queryDataSourceMapper,
  queryLogs: queryLogsMapper,
  queryTraces: queryTracesMapper,
  renderPanel: renderPanelMapper,
  updateAlertRule: updateAlertRuleMapper,
  updateHttpSyntheticCheck: updateHttpSyntheticCheckMapper,
  createAnnotation: createAnnotationMapper,
  listAnnotations: listAnnotationsMapper,
  deleteAnnotation: deleteAnnotationMapper,
  createSilence: createSilenceMapper,
  deleteSilence: deleteSilenceMapper,
  deleteHttpSyntheticCheck: deleteHttpSyntheticCheckMapper,
  getSilence: getSilenceMapper,
  listSilences: listSilencesMapper,
  declareIncident: declareIncidentMapper,
  declareDrill: declareDrillMapper,
  getIncident: getIncidentMapper,
  updateIncident: updateIncidentMapper,
  resolveIncident: resolveIncidentMapper,
  addIncidentActivity: addIncidentActivityMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createAlertRule: buildActionStateRegistry("created"),
  createHttpSyntheticCheck: buildActionStateRegistry("created"),
  deleteAlertRule: buildActionStateRegistry("deleted"),
  deleteHttpSyntheticCheck: buildActionStateRegistry("deleted"),
  getAlertRule: buildActionStateRegistry("fetched"),
  getDashboard: buildActionStateRegistry("fetched"),
  getHttpSyntheticCheck: GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY,
  listAlertRules: buildActionStateRegistry("listed"),
  queryDataSource: buildActionStateRegistry("queried"),
  queryLogs: buildActionStateRegistry("queried"),
  queryTraces: buildActionStateRegistry("queried"),
  renderPanel: buildActionStateRegistry("rendered"),
  updateAlertRule: buildActionStateRegistry("updated"),
  updateHttpSyntheticCheck: buildActionStateRegistry("updated"),
  createAnnotation: buildActionStateRegistry("created"),
  listAnnotations: buildActionStateRegistry("listed"),
  deleteAnnotation: buildActionStateRegistry("deleted"),
  createSilence: buildActionStateRegistry("created"),
  deleteSilence: buildActionStateRegistry("deleted"),
  getSilence: getSilenceEventStateRegistry,
  listSilences: buildActionStateRegistry("listed"),
  declareIncident: buildActionStateRegistry("declared"),
  declareDrill: buildActionStateRegistry("declared"),
  getIncident: buildActionStateRegistry("fetched"),
  updateIncident: buildActionStateRegistry("updated"),
  resolveIncident: buildActionStateRegistry("resolved"),
  addIncidentActivity: buildActionStateRegistry("added"),
};
