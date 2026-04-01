import type { ComponentBaseMapper, CustomFieldRenderer, TriggerRenderer, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createDashboardShareLinkMapper } from "./create_dashboard_share_link";
import { getDashboardMapper } from "./get_dashboard";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import { queryDataSourceMapper } from "./query_data_source";
import { renderPanelMapper } from "./render_panel";
import { searchDashboardsMapper } from "./search_dashboards";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDashboardShareLink: createDashboardShareLinkMapper,
  getDashboard: getDashboardMapper,
  queryDataSource: queryDataSourceMapper,
  renderPanel: renderPanelMapper,
  searchDashboards: searchDashboardsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onAlertFiring: onAlertFiringTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDashboardShareLink: buildActionStateRegistry("created"),
  getDashboard: buildActionStateRegistry("fetched"),
  queryDataSource: buildActionStateRegistry("queried"),
  renderPanel: buildActionStateRegistry("rendered"),
  searchDashboards: buildActionStateRegistry("listed"),
};
