import { describe, expect, it } from "vitest";

import { getDashboardMapper } from "./get_dashboard";
import { renderPanelMapper } from "./render_panel";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

function makeNode(componentName: string, configuration: unknown = {}, metadata: unknown = {}): NodeInfo {
  return {
    id: `${componentName}-node`,
    name: componentName,
    componentName: `grafana.${componentName}`,
    isCollapsed: false,
    configuration,
    metadata,
  };
}

function makeExecution(outputs?: { default?: OutputPayload[] }): ExecutionInfo {
  const now = new Date().toISOString();

  return {
    id: "exec-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_FINISHED" as ExecutionInfo["state"],
    result: "RESULT_SUCCEEDED" as ExecutionInfo["result"],
    resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs,
  };
}

function makeComponentContext(node: NodeInfo, lastExecutions: ExecutionInfo[] = []): ComponentBaseContext {
  return {
    nodes: [],
    node,
    componentDefinition: {
      name: node.componentName.replace("grafana.", ""),
      label: node.name,
      description: "",
      icon: "grafana",
      color: "blue",
    },
    lastExecutions,
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

function makeExecutionContext(node: NodeInfo, outputs?: { default?: OutputPayload[] }): ExecutionDetailsContext {
  return {
    nodes: [node],
    node,
    execution: makeExecution(outputs),
  };
}

describe("grafana dashboard mappers", () => {
  it("getDashboardMapper omits trigger event section when execution has no root event", () => {
    const node = makeNode("getDashboard", { dashboard: "dash-prod" }, { dashboardTitle: "Production Overview" });
    const props = getDashboardMapper.props(makeComponentContext(node, [makeExecution()]));

    expect(props.eventSections).toEqual([]);
  });

  it("renderPanelMapper omits trigger event section when execution has no root event", () => {
    const node = makeNode("renderPanel", { dashboard: "dash-prod", panel: 7 });
    const props = renderPanelMapper.props(makeComponentContext(node, [makeExecution()]));

    expect(props.eventSections).toEqual([]);
  });

  it("getDashboardMapper uses dashboard metadata fallback and handles sparse outputs", () => {
    const node = makeNode("getDashboard", { dashboard: "dash-prod" }, { dashboardTitle: "Production Overview" });

    const props = getDashboardMapper.props(makeComponentContext(node));
    const details = getDashboardMapper.getExecutionDetails(makeExecutionContext(node));

    expect(props.metadata).toEqual([{ icon: "layout-dashboard", label: "Production Overview" }]);
    expect(details.Response).toBe("No data returned");
    expect(details["Fetched At"]).toBeDefined();
  });

  it("getDashboardMapper summarizes returned dashboard details", () => {
    const node = makeNode("getDashboard", { dashboard: "dash-prod" });
    const details = getDashboardMapper.getExecutionDetails(
      makeExecutionContext(node, {
        default: [
          {
            type: "json",
            timestamp: new Date().toISOString(),
            data: {
              title: "Production Overview",
              url: "/d/dash-prod/production-overview",
              folderTitle: "Platform",
              panels: [{ id: 1 }, { id: 2 }],
            },
          },
        ],
      }),
    );

    expect(details.Title).toBe("Production Overview");
    expect(details["Dashboard URL"]).toBe("/d/dash-prod/production-overview");
    expect(details.Folder).toBe("Platform");
    expect(details.Panels).toBe("2 panels");
  });

  it("renderPanelMapper uses renamed dashboard and panel fields in metadata and no-throw details", () => {
    const node = makeNode(
      "renderPanel",
      { dashboard: "dash-prod", panel: 7, from: "now-1h", to: "now", width: 1200, height: 600 },
      { dashboardTitle: "Production Overview", panelTitle: "Error Rate", panelLabel: "#7 Error Rate" },
    );

    const props = renderPanelMapper.props(makeComponentContext(node));
    const details = renderPanelMapper.getExecutionDetails(makeExecutionContext(node));

    expect(props.metadata).toEqual([
      { icon: "layout-dashboard", label: "Production Overview" },
      { icon: "hash", label: "#7 Error Rate" },
      { icon: "clock-3", label: "now-1h -> now" },
    ]);
    expect(details.Response).toBe("No data returned");
    expect(details["Rendered At"]).toBeDefined();
  });

  it("renderPanelMapper summarizes render output", () => {
    const node = makeNode("renderPanel", { dashboard: "dash-prod", panel: 7 });
    const details = renderPanelMapper.getExecutionDetails(
      makeExecutionContext(node, {
        default: [
          {
            type: "json",
            timestamp: new Date().toISOString(),
            data: {
              url: "https://grafana.example.com/render/d-solo/dash-prod/production-overview?panelId=7",
              dashboard: "dash-prod",
              panel: 7,
            },
          },
        ],
      }),
    );

    expect(details.Dashboard).toBe("dash-prod");
    expect(details.Panel).toBe("7");
    expect(details.URL).toContain("panelId=7");
  });
});
