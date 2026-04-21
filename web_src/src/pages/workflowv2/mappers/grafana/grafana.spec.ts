import { describe, expect, it } from "vitest";
import { getDashboardMapper } from "./get_dashboard";
import { renderPanelMapper } from "./render_panel";
import { queryDataSourceMapper } from "./query_data_source";
import { onAlertFiringTriggerRenderer } from "./on_alert_firing";
import type {
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
  TriggerRendererContext,
} from "../types";

// ===== Test Helpers =====

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Grafana",
    componentName: "grafana",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  const now = new Date().toISOString();

  return {
    id: "exec-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

function buildTriggerRendererContext(overrides?: Partial<TriggerRendererContext>): TriggerRendererContext {
  return {
    node: buildNode(),
    definition: { name: "", label: "On Alert Firing", description: "", icon: "", color: "blue" },
    lastEvent: undefined,
    ...overrides,
  };
}

/** Minimal fields required by `EventInfo` for trigger renderer tests */
const defaultTriggerEvent = {
  nodeId: "node-trigger-1",
  type: "grafana.alert.firing",
} as const;

// ===== Get Dashboard Mapper Tests =====

describe("getDashboardMapper.getExecutionDetails", () => {
  it("returns no data when outputs are empty", () => {
    const node = buildNode({ componentName: "getDashboard" });
    const ctx: ExecutionDetailsContext = { nodes: [node], node, execution: buildExecution({ outputs: {} }) };

    const details = getDashboardMapper.getExecutionDetails(ctx);
    expect(details.Response).toBe("No data returned");
    expect(details["Fetched At"]).toBeDefined();
  });

  it("extracts dashboard details from output payload", () => {
    const node = buildNode({ componentName: "getDashboard" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "json",
              timestamp: new Date().toISOString(),
              data: {
                uid: "dash-123",
                title: "Test Dashboard",
                url: "/d/dash-123/test",
                folderTitle: "Monitoring",
                panels: [{ id: 1 }, { id: 2 }],
              },
            },
          ],
        },
      }),
    };

    const details = getDashboardMapper.getExecutionDetails(ctx);
    expect(details.Title).toBe("Test Dashboard");
    expect(details["Dashboard URL"]).toBe("/d/dash-123/test");
    expect(details.Folder).toBe("Monitoring");
    expect(details.Panels).toBe("2 panels");
  });

  it("prefers folder title over folder UID when both are present", () => {
    const node = buildNode({ componentName: "getDashboard" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "json",
              timestamp: new Date().toISOString(),
              data: {
                title: "Prod",
                folderTitle: "Platform",
                folder: "fdg4m1rt63hj8q",
              },
            },
          ],
        },
      }),
    };

    expect(getDashboardMapper.getExecutionDetails(ctx).Folder).toBe("Platform");
  });

  it("returns 0 panels when panels array is absent", () => {
    const node = buildNode({ componentName: "getDashboard" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: { default: [{ type: "json", timestamp: new Date().toISOString(), data: { title: "Minimal" } }] },
      }),
    };

    expect(getDashboardMapper.getExecutionDetails(ctx).Panels).toBe("0 panels");
  });
});

describe("getDashboardMapper.subtitle", () => {
  it("returns time ago when createdAt is present", () => {
    const node = buildNode({ componentName: "getDashboard" });
    const ctx: SubtitleContext = {
      node,
      execution: buildExecution({ createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString() }),
    };

    expect(getDashboardMapper.subtitle(ctx)).not.toBe("-");
  });

  it("returns dash when createdAt is missing", () => {
    const node = buildNode({ componentName: "getDashboard" });
    const ctx: SubtitleContext = {
      node,
      execution: { ...buildExecution({}), createdAt: undefined as unknown as string },
    };

    expect(getDashboardMapper.subtitle(ctx)).toBe("-");
  });
});

// ===== Render Panel Mapper Tests =====

describe("renderPanelMapper.getExecutionDetails", () => {
  it("returns no data when outputs are empty", () => {
    const node = buildNode({ componentName: "renderPanel" });
    const ctx: ExecutionDetailsContext = { nodes: [node], node, execution: buildExecution({ outputs: {} }) };

    const details = renderPanelMapper.getExecutionDetails(ctx);
    expect(details.Response).toBe("No data returned");
    expect(details["Rendered At"]).toBeDefined();
  });

  it("extracts dashboard and panel from output", () => {
    const node = buildNode({ componentName: "renderPanel" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "json",
              timestamp: new Date().toISOString(),
              data: { url: "http://grafana.local/render", dashboard: "dash-123", panel: 1 },
            },
          ],
        },
      }),
    };

    const details = renderPanelMapper.getExecutionDetails(ctx);
    expect(details.Dashboard).toBe("dash-123");
    expect(details.Panel).toBe("1");
    expect(details.URL).toBeDefined();
  });

  it("omits dashboard and panel when absent in output", () => {
    const node = buildNode({ componentName: "renderPanel" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          default: [
            { type: "json", timestamp: new Date().toISOString(), data: { url: "http://grafana.local/render" } },
          ],
        },
      }),
    };

    const details = renderPanelMapper.getExecutionDetails(ctx);
    expect(details.Dashboard).toBeUndefined();
    expect(details.Panel).toBeUndefined();
    expect(details.URL).toBeDefined();
  });
});

// ===== Query Data Source Mapper Tests =====

describe("queryDataSourceMapper.getExecutionDetails", () => {
  it("returns no data when default outputs are empty", () => {
    const node = buildNode({ componentName: "queryDataSource" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({ outputs: { default: [] } }),
    };

    const details = queryDataSourceMapper.getExecutionDetails(ctx);
    expect(details.Response).toBe("No data returned");
  });

  it("includes datasource and query from configuration", () => {
    const node = buildNode({
      componentName: "queryDataSource",
      configuration: { dataSource: "prom-uid-123", query: "node_cpu_seconds_total", format: "table" },
    });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({ outputs: { default: [] } }),
    };

    const details = queryDataSourceMapper.getExecutionDetails(ctx);
    expect(details["Data Source"]).toBe("prom-uid-123");
    expect(details.Query).toBe("node_cpu_seconds_total");
    expect(details.Format).toBe("table");
  });

  it("processes query results with frames and fields", () => {
    const node = buildNode({ componentName: "queryDataSource" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "json",
              timestamp: new Date().toISOString(),
              data: {
                results: {
                  A: {
                    frames: [
                      {
                        schema: { fields: [{ name: "time" }, { name: "value" }] },
                        data: {
                          values: [
                            ["t1", "t2"],
                            [1, 2],
                          ],
                        },
                      },
                    ],
                  },
                },
              },
            },
          ],
        },
      }),
    };

    const details = queryDataSourceMapper.getExecutionDetails(ctx);
    expect(details["Result Ref IDs"]).toBe("A");
    expect(details["Frame Count"]).toBe("1");
    expect(details.Fields).toContain("time");
  });

  it("handles empty results object", () => {
    const node = buildNode({ componentName: "queryDataSource" });
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: buildExecution({
        outputs: { default: [{ type: "json", timestamp: new Date().toISOString(), data: { results: {} } }] },
      }),
    };

    expect(queryDataSourceMapper.getExecutionDetails(ctx).Results).toBe("No results");
  });
});

// ===== On Alert Firing Trigger Renderer Tests =====

describe("onAlertFiringTriggerRenderer.subtitle", () => {
  it("returns a subtitle when event data is present", () => {
    const ctx = {
      event: { ...defaultTriggerEvent, id: "e1", createdAt: new Date().toISOString(), data: undefined },
    };
    expect(onAlertFiringTriggerRenderer.subtitle(ctx)).toBeTruthy();
  });

  it("returns a subtitle node for a firing event", () => {
    const ctx = {
      event: {
        ...defaultTriggerEvent,
        id: "e1",
        createdAt: new Date().toISOString(),
        runTitle: "High CPU",
        data: { title: "High CPU", status: "firing" },
      },
    };
    expect(onAlertFiringTriggerRenderer.subtitle(ctx)).toBeTruthy();
  });

  it("returns a subtitle node when labels are present", () => {
    const ctx = {
      event: {
        ...defaultTriggerEvent,
        id: "e1",
        createdAt: new Date().toISOString(),
        data: { commonLabels: { alertname: "DiskWarn" }, status: "firing" },
      },
    };
    expect(onAlertFiringTriggerRenderer.subtitle(ctx)).toBeTruthy();
  });

  it("returns a subtitle node when alerts are present", () => {
    const ctx = {
      event: {
        ...defaultTriggerEvent,
        id: "e1",
        createdAt: new Date().toISOString(),
        data: { alerts: [{ labels: { alertname: "MemoryLeak" } }], status: "firing" },
      },
    };
    expect(onAlertFiringTriggerRenderer.subtitle(ctx)).toBeTruthy();
  });
});

describe("onAlertFiringTriggerRenderer.getRootEventValues", () => {
  it("returns all alert fields from event data", () => {
    const ctx = {
      event: {
        ...defaultTriggerEvent,
        id: "e1",
        createdAt: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
        data: {
          title: "API Error Rate High",
          status: "firing",
          ruleUid: "api_error_rate_high",
          ruleId: 42,
          orgId: 1,
          externalURL: "https://grafana.example.com",
        },
      },
    };

    const values = onAlertFiringTriggerRenderer.getRootEventValues(ctx);
    expect(values.Status).toBe("firing");
    expect(values["Alert Name"]).toBe("API Error Rate High");
    expect(values["Rule UID"]).toBe("api_error_rate_high");
    expect(values["Rule ID"]).toBe("42");
    expect(values["Org ID"]).toBe("1");
    expect(values["External URL"]).toBe("https://grafana.example.com");
    expect(values["Triggered At"]).toBeDefined();
  });

  it("returns dashes for missing optional fields", () => {
    const ctx = {
      event: {
        ...defaultTriggerEvent,
        id: "e1",
        createdAt: new Date().toISOString(),
        data: { status: "firing" },
      },
    };
    const values = onAlertFiringTriggerRenderer.getRootEventValues(ctx);
    expect(values.Status).toBe("firing");
    expect(values["Alert Name"]).toBe("-");
    expect(values["Rule UID"]).toBe("-");
  });

  it("defaults status to firing when absent", () => {
    const ctx = {
      event: { ...defaultTriggerEvent, id: "e1", createdAt: new Date().toISOString(), data: {} },
    };
    expect(onAlertFiringTriggerRenderer.getRootEventValues(ctx).Status).toBe("firing");
  });
});

describe("onAlertFiringTriggerRenderer.getTriggerProps", () => {
  it("builds props from node and configured alert names", () => {
    const ctx = buildTriggerRendererContext({
      node: buildNode({
        name: "Grafana Alert",
        componentName: "onAlertFiring",
        configuration: {
          alertNames: [
            { type: "equals", value: "HighCPU" },
            { type: "matches", value: ".*Error.*" },
          ],
        },
      }),
    });

    const props = onAlertFiringTriggerRenderer.getTriggerProps(ctx);
    expect(props.title).toBe("Grafana Alert");
    expect(props.metadata).toBeDefined();
  });

  it("includes last event data when available", () => {
    const ctx = buildTriggerRendererContext({
      node: buildNode({ name: "Grafana Alert", componentName: "onAlertFiring" }),
      lastEvent: {
        id: "event-1",
        createdAt: new Date().toISOString(),
        data: { title: "Recent Alert", status: "firing" },
        nodeId: "node-1",
        type: "grafana.alert.firing",
        runTitle: "Recent Alert",
      },
    });

    const props = onAlertFiringTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData?.subtitle).toBeDefined();
  });
});
