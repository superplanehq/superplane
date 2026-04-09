import type { ComponentBaseMapper, CustomFieldRenderer, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getDashboardMapper } from "./get_dashboard";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { renderPanelMapper } from "./render_panel";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getDashboard: getDashboardMapper,
  queryDataSource: queryDataSourceMapper,
  renderPanel: renderPanelMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getDashboard: buildActionStateRegistry("fetched"),
  queryDataSource: buildActionStateRegistry("queried"),
  renderPanel: buildActionStateRegistry("rendered"),
};
