import type { ComponentBaseMapper, CustomFieldRenderer, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createAlertRuleMapper } from "./create_alert_rule";
import { deleteAlertRuleMapper } from "./delete_alert_rule";
import { getAlertRuleMapper } from "./get_alert_rule";
import { listAlertRulesMapper } from "./list_alert_rules";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { updateAlertRuleMapper } from "./update_alert_rule";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createAlertRule: createAlertRuleMapper,
  deleteAlertRule: deleteAlertRuleMapper,
  getAlertRule: getAlertRuleMapper,
  listAlertRules: listAlertRulesMapper,
  queryDataSource: queryDataSourceMapper,
  updateAlertRule: updateAlertRuleMapper,
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
  updateAlertRule: buildActionStateRegistry("updated"),
};
