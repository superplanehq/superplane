import { describe, expect, it } from "vitest";

import { queryDataSourceMapper } from "./query_data_source";
import { queryLogsMapper } from "./query_logs";
import { queryTracesMapper } from "./query_traces";
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
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
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

function makeComponentContext(node: NodeInfo): ComponentBaseContext {
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
    lastExecutions: [],
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

describe("grafana data source mappers", () => {
  it("queryDataSourceMapper uses renamed dataSource field in metadata and execution details", () => {
    const node = makeNode("queryDataSource", {
      dataSource: "prom-main",
      query: "up",
      format: "table",
    });

    const props = queryDataSourceMapper.props(makeComponentContext(node));
    const details = queryDataSourceMapper.getExecutionDetails(makeExecutionContext(node));

    expect(props.metadata).toEqual([
      { icon: "database", label: "Data Source: prom-main" },
      { icon: "code", label: "up" },
      { icon: "funnel", label: "Format: table" },
    ]);
    expect(details["Data Source"]).toBe("prom-main");
    expect(details.Query).toBe("up");
    expect(details.Format).toBe("table");
    expect(details.Response).toBe("No data returned");
  });

  it("queryLogsMapper does not throw with sparse outputs and uses renamed dataSource field", () => {
    const node = makeNode("queryLogs", {
      dataSource: "loki-main",
      query: `{app="api"} |= "error"`,
    });
    const context = makeExecutionContext(node, { default: [] });

    expect(() => queryLogsMapper.getExecutionDetails(context)).not.toThrow();
    expect(queryLogsMapper.props(makeComponentContext(node)).metadata).toEqual([
      { icon: "database", label: "Data Source: loki-main" },
      { icon: "code", label: `{app="api"} |= "error"` },
    ]);
    expect(queryLogsMapper.getExecutionDetails(context)["Data Source"]).toBe("loki-main");
    expect(queryLogsMapper.getExecutionDetails(context)["Log Lines"]).toBe("0");
  });

  it("queryTracesMapper does not throw with sparse outputs and uses renamed dataSource field", () => {
    const node = makeNode("queryTraces", {
      dataSource: "tempo-main",
      query: "{ .http.status_code = 500 }",
    });
    const context = makeExecutionContext(node, { default: [] });

    expect(() => queryTracesMapper.getExecutionDetails(context)).not.toThrow();
    expect(queryTracesMapper.props(makeComponentContext(node)).metadata).toEqual([
      { icon: "database", label: "Data Source: tempo-main" },
      { icon: "code", label: "{ .http.status_code = 500 }" },
    ]);
    expect(queryTracesMapper.getExecutionDetails(context)["Data Source"]).toBe("tempo-main");
    expect(queryTracesMapper.getExecutionDetails(context).Traces).toBe("0");
  });

  it("queryLogsMapper reports no data when output exists but payload data is missing", () => {
    const node = makeNode("queryLogs", { dataSource: "loki-main", query: "{}" });
    const context = makeExecutionContext(node, {
      default: [
        {
          type: "grafana.logs.result",
          timestamp: new Date().toISOString(),
          data: null,
        },
      ],
    });

    expect(queryLogsMapper.getExecutionDetails(context)["Log Lines"]).toBe("No data returned");
  });

  it("queryTracesMapper reports no data when output exists but payload data is missing", () => {
    const node = makeNode("queryTraces", { dataSource: "tempo-main", query: "{}" });
    const context = makeExecutionContext(node, {
      default: [
        {
          type: "grafana.traces.result",
          timestamp: new Date().toISOString(),
          data: null,
        },
      ],
    });

    expect(queryTracesMapper.getExecutionDetails(context).Traces).toBe("No data returned");
  });
});

function makeGrafanaQueryResponse(rowsPerFrame: number[]): Record<string, unknown> {
  return {
    results: {
      A: {
        frames: rowsPerFrame.map((n) => ({
          data: { values: [Array.from({ length: n }, (_, i) => i)] },
        })),
      },
    },
  };
}

describe("queryLogsMapper", () => {
  it("returns empty metadata when configuration is missing", () => {
    const node = makeNode("queryLogs");

    expect(queryLogsMapper.props(makeComponentContext(node)).metadata).toEqual([]);
  });

  it("truncates long queries to 50 chars in metadata", () => {
    const longQuery = `{app="api"} |= "` + "e".repeat(60) + '"';
    const node = makeNode("queryLogs", { dataSource: "loki", query: longQuery });

    const metadata = queryLogsMapper.props(makeComponentContext(node)).metadata ?? [];
    const label = metadata[1].label as string;

    expect(label.length).toBeLessThanOrEqual(53); // 50 + "..."
    expect(label.endsWith("...")).toBe(true);
  });

  it("truncates long queries to 80 chars in execution details", () => {
    const longQuery = "{app=" + "a".repeat(100) + "}";
    const node = makeNode("queryLogs", { dataSource: "loki", query: longQuery });
    const details = queryLogsMapper.getExecutionDetails(makeExecutionContext(node));

    expect(details["Query"].length).toBeLessThanOrEqual(83); // 80 + "..."
    expect(details["Query"].endsWith("...")).toBe(true);
  });

  it("returns '0' log lines when outputs is undefined", () => {
    const node = makeNode("queryLogs", { dataSource: "loki", query: "{}" });
    const details = queryLogsMapper.getExecutionDetails(makeExecutionContext(node));

    expect(details["Log Lines"]).toBe("0");
  });

  it("counts log lines from response data frames", () => {
    const node = makeNode("queryLogs", { dataSource: "loki", query: "{}" });
    const responseData = makeGrafanaQueryResponse([5, 3]);
    const context = makeExecutionContext(node, {
      default: [{ type: "grafana.logs.result", timestamp: new Date().toISOString(), data: responseData }],
    });

    expect(queryLogsMapper.getExecutionDetails(context)["Log Lines"]).toBe("8");
  });

  it("updates 'Queried At' from payload timestamp", () => {
    const payloadTs = "2024-06-01T12:00:00.000Z";
    const node = makeNode("queryLogs", { dataSource: "loki", query: "{}" });
    const context = makeExecutionContext(node, {
      default: [{ type: "grafana.logs.result", timestamp: payloadTs, data: makeGrafanaQueryResponse([1]) }],
    });

    expect(queryLogsMapper.getExecutionDetails(context)["Queried At"]).not.toBe("-");
  });
});

describe("queryTracesMapper", () => {
  it("returns empty metadata when configuration is missing", () => {
    const node = makeNode("queryTraces");

    expect(queryTracesMapper.props(makeComponentContext(node)).metadata).toEqual([]);
  });

  it("truncates long queries to 50 chars in metadata", () => {
    const longQuery = "{ .http.method = " + '"' + "G".repeat(60) + '" }';
    const node = makeNode("queryTraces", { dataSource: "tempo", query: longQuery });

    const metadata = queryTracesMapper.props(makeComponentContext(node)).metadata ?? [];
    const label = metadata[1].label as string;

    expect(label.length).toBeLessThanOrEqual(53);
    expect(label.endsWith("...")).toBe(true);
  });

  it("truncates long queries to 80 chars in execution details", () => {
    const longQuery = "{ .service.name = " + '"' + "x".repeat(100) + '" }';
    const node = makeNode("queryTraces", { dataSource: "tempo", query: longQuery });
    const details = queryTracesMapper.getExecutionDetails(makeExecutionContext(node));

    expect(details["Query"].length).toBeLessThanOrEqual(83);
    expect(details["Query"].endsWith("...")).toBe(true);
  });

  it("returns '0' traces when outputs is undefined", () => {
    const node = makeNode("queryTraces", { dataSource: "tempo", query: "{}" });
    const details = queryTracesMapper.getExecutionDetails(makeExecutionContext(node));

    expect(details["Traces"]).toBe("0");
  });

  it("counts traces from response data frames", () => {
    const node = makeNode("queryTraces", { dataSource: "tempo", query: "{}" });
    const responseData = makeGrafanaQueryResponse([10]);
    const context = makeExecutionContext(node, {
      default: [{ type: "grafana.traces.result", timestamp: new Date().toISOString(), data: responseData }],
    });

    expect(queryTracesMapper.getExecutionDetails(context)["Traces"]).toBe("10");
  });

  it("updates 'Queried At' from payload timestamp", () => {
    const payloadTs = "2024-06-01T12:00:00.000Z";
    const node = makeNode("queryTraces", { dataSource: "tempo", query: "{}" });
    const context = makeExecutionContext(node, {
      default: [{ type: "grafana.traces.result", timestamp: payloadTs, data: makeGrafanaQueryResponse([1]) }],
    });

    expect(queryTracesMapper.getExecutionDetails(context)["Queried At"]).not.toBe("-");
  });
});
