import { describe, expect, it } from "vitest";

import { deleteSilenceMapper } from "./delete_silence";
import { getSilenceMapper } from "./get_silence";
import { queryDataSourceMapper } from "./query_data_source";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

function buildNode(componentName: string, overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Grafana Resource Mapper",
    componentName,
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
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

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "grafana.result",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildComponentContext(componentName: string, nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(componentName, nodeOverrides);

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: componentName,
      label: componentName,
      description: "",
      icon: "bolt",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

function buildExecutionContext(
  componentName: string,
  overrides?: { node?: Partial<NodeInfo>; execution?: Partial<ExecutionInfo> },
): ExecutionDetailsContext {
  const node = buildNode(componentName, overrides?.node);

  return {
    nodes: [node],
    node,
    execution: buildExecution(overrides?.execution),
  };
}

describe("queryDataSourceMapper", () => {
  it("uses configuration.dataSource in metadata", () => {
    const props = queryDataSourceMapper.props(
      buildComponentContext("grafana.queryDataSource", {
        configuration: {
          dataSource: "prometheus-main",
          query: "up",
          format: "time_series",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "Data Source: prometheus-main" }),
        expect.objectContaining({ label: "up" }),
        expect.objectContaining({ label: "Format: time_series" }),
      ]),
    );
  });

  it("uses configuration.dataSource in execution details", () => {
    const details = queryDataSourceMapper.getExecutionDetails(
      buildExecutionContext("grafana.queryDataSource", {
        node: {
          configuration: {
            dataSource: "prometheus-main",
            query: "up",
          },
        },
        execution: {
          outputs: {
            default: [buildOutput({})],
          },
        },
      }),
    );

    expect(details["Data Source"]).toBe("prometheus-main");
  });
});

describe("silence selection mappers", () => {
  it("getSilenceMapper uses configuration.silence in metadata", () => {
    const props = getSilenceMapper.props(
      buildComponentContext("grafana.getSilence", {
        configuration: {
          silence: "silence-123",
        },
      }),
    );

    expect(props.metadata).toEqual([expect.objectContaining({ label: "silence-123" })]);
  });

  it("deleteSilenceMapper uses configuration.silence in metadata", () => {
    const props = deleteSilenceMapper.props(
      buildComponentContext("grafana.deleteSilence", {
        configuration: {
          silence: "silence-456",
        },
      }),
    );

    expect(props.metadata).toEqual([expect.objectContaining({ label: "silence-456" })]);
  });

  it("silence mappers still tolerate missing outputs", () => {
    const getCtx = buildExecutionContext("grafana.getSilence", {
      node: { configuration: { silence: "silence-123" } },
      execution: { outputs: undefined },
    });
    const deleteCtx = buildExecutionContext("grafana.deleteSilence", {
      node: { configuration: { silence: "silence-456" } },
      execution: { outputs: undefined },
    });

    expect(() => getSilenceMapper.getExecutionDetails(getCtx)).not.toThrow();
    expect(() => deleteSilenceMapper.getExecutionDetails(deleteCtx)).not.toThrow();
  });
});
