/**
 * Default content templates for each panel kind. Extracted from
 * `panelTypes.ts` to keep that module under the shared lint budget and
 * to lower the branching complexity of `templateForPanelType` (the
 * previous `switch` triggered the eslint `complexity` rule once the
 * `board` case was added).
 *
 * Kept in lockstep with `panelTypes.ts` — the type-safe entry point
 * `templateForPanelType` still lives there and just dispatches through
 * this table.
 */

import { templateForBoardPanel } from "./boardPanelContent";
import { templateForNodesPanel } from "./nodesPanelContent";
import type { PanelType } from "./panelTypes";
import type { WidgetChartRender, WidgetNumberRender, WidgetScorecardRender, WidgetTableRender } from "./widget/types";

const DEFAULT_TABLE_RENDER: WidgetTableRender = {
  kind: "table",
  columns: [],
};

const DEFAULT_CHART_RENDER: WidgetChartRender = {
  kind: "chart",
  type: "bar",
  xField: "status",
  series: [{ label: "Count" }],
};

const DEFAULT_NUMBER_RENDER: WidgetNumberRender = {
  kind: "number",
  aggregation: "count",
  label: "Runs",
};

// `count` needs no field, so a fresh scorecard validates before the author
// picks a data source or switches to a field-backed aggregation.
const DEFAULT_SCORECARD_RENDER: WidgetScorecardRender = {
  kind: "scorecard",
  aggregation: "count",
  better: "up",
  showChange: "both",
  changeCaption: "vs previous",
};

type TemplateBuilder = (defaultTitle?: string) => Record<string, unknown>;

const TEMPLATE_BUILDERS: Record<PanelType, TemplateBuilder> = {
  markdown: (t) => ({ title: t ?? "", body: "", variables: [] }),
  html: (t) => ({ title: t ?? "", body: "", variables: [] }),
  node: (t) => ({ title: t ?? "", node: "", showRun: false }),
  nodes: (t) => ({ ...templateForNodesPanel(t) }),
  table: (t) => ({ title: t ?? "", dataSource: { kind: "memory", namespace: "" }, render: DEFAULT_TABLE_RENDER }),
  board: (t) => ({ ...templateForBoardPanel(t) }),
  chart: (t) => ({ title: t ?? "", dataSource: { kind: "executions", limit: 100 }, render: DEFAULT_CHART_RENDER }),
  number: (t) => ({ title: t ?? "", dataSource: { kind: "runs", limit: 100 }, render: DEFAULT_NUMBER_RENDER }),
  scorecard: (t) => ({
    title: t ?? "",
    dataSource: { kind: "memory", namespace: "" },
    render: DEFAULT_SCORECARD_RENDER,
  }),
};

export function buildTemplateForPanelType(type: PanelType, defaultTitle?: string): Record<string, unknown> {
  return TEMPLATE_BUILDERS[type](defaultTitle);
}
