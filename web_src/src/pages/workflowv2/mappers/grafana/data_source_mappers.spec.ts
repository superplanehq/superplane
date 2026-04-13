import { describe, expect, it } from "vitest";

import { getDataSourceMapper } from "./get_data_source";
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
      invokeNodeExecutionAction: async () => {},
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
  it("getDataSourceMapper uses renamed dataSource configuration for metadata fallback", () => {
    const node = makeNode("getDataSource", { dataSource: "loki-main" });

    const metadata = getDataSourceMapper.props(makeComponentContext(node)).metadata ?? [];

    expect(metadata).toEqual([{ icon: "database", label: "Data Source: loki-main" }]);
  });

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
});
