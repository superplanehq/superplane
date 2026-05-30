import { describe, expect, it } from "vitest";
import { renderOperationMapper } from "./operations";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Get Metrics",
  componentName: "render.getMetrics",
  isCollapsed: false,
};

const DEFINITION = {
  name: "render.getMetrics",
  label: "Get Metrics",
  description: "",
  icon: "activity",
  color: "gray",
};

function makePropsContext(configuration: Record<string, unknown> | undefined): ComponentBaseContext {
  return {
    nodes: [],
    node: { ...NODE, configuration },
    componentDefinition: DEFINITION,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

function makeDetailsContext(outputData?: Record<string, unknown>): ExecutionDetailsContext {
  const execution: ExecutionInfo = {
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
    outputs: outputData ? { default: [{ data: outputData, type: "render.metrics" }] } : undefined,
  };
  return { nodes: [], node: NODE, execution };
}

describe("renderOperationMapper.props", () => {
  it("renders resource and metric metadata", () => {
    const props = renderOperationMapper.props!(
      makePropsContext({
        resources: ["srv-123", "dpg-123"],
        metricTypes: ["cpu", "memory"],
        limit: 20,
      }),
    );

    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Resources: 2" })]));
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ label: "Metrics: cpu, memory" })]),
    );
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Limit: 20" })]));
  });
});

describe("renderOperationMapper.getExecutionDetails", () => {
  it("returns normalized details from the default output", () => {
    const details = renderOperationMapper.getExecutionDetails!(
      makeDetailsContext({
        serviceId: "srv-123",
        status: "updated",
        count: 3,
        errorCount: 1,
        autoDeploy: "no",
        resources: ["srv-123", "srv-456"],
      }),
    );

    expect(details["Service ID"]).toBe("srv-123");
    expect(details.Status).toBe("updated");
    expect(details.Count).toBe("3");
    expect(details["Error Count"]).toBe("1");
    expect(details["Auto Deploy"]).toBe("no");
    expect(details.Resources).toBe("srv-123, srv-456");
  });
});
