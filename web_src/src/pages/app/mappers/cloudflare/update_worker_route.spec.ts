import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
} from "../types";
import { updateWorkerRouteMapper } from "./update_worker_route";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.updateWorkerRoute",
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

function buildDetailsCtx(overrides?: { execution?: Partial<ExecutionInfo> }): ExecutionDetailsContext {
  const node = buildNode();
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.updateWorkerRoute",
  label: "Update Worker Route",
  description: "",
  icon: "cloud",
  color: "orange",
};

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

describe("updateWorkerRouteMapper.getExecutionDetails", () => {
  it("reads route from output data", () => {
    const data = { route: { id: "r1", pattern: "ex.com/*", script: "w" } };
    const outputs = {
      default: [{ type: "cloudflare.workerRoute.created", timestamp: new Date().toISOString(), data }],
    };
    const details = updateWorkerRouteMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs } }));
    expect(details["Pattern"]).toBe("ex.com/*");
    expect(details["Script"]).toBe("w");
  });
});

describe("updateWorkerRouteMapper.props metadata", () => {
  it("shows create when routeId is absent", () => {
    const props = updateWorkerRouteMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { pattern: "ex.com/*", workerScript: "w" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "route", label: "ex.com/*" },
      { icon: "code", label: "w" },
      { icon: "plus", label: "Create" },
    ]);
  });

  it("shows update when routeId is set", () => {
    const props = updateWorkerRouteMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { pattern: "ex.com/*", workerScript: "w", routeId: "rid" },
        }),
      }),
    );
    expect(props.metadata?.[2]).toEqual({ icon: "edit", label: "Update" });
  });
});

describe("eventStateRegistry.updateWorkerRoute", () => {
  it("returns created when payload type is created", () => {
    const execution = buildExecution({
      outputs: {
        default: [{ type: "cloudflare.workerRoute.created", timestamp: new Date().toISOString(), data: {} }],
      },
    });
    expect(eventStateRegistry.updateWorkerRoute.getState(execution)).toBe("created");
  });

  it("returns updated when payload type is updated", () => {
    const execution = buildExecution({
      outputs: {
        default: [{ type: "cloudflare.workerRoute.updated", timestamp: new Date().toISOString(), data: {} }],
      },
    });
    expect(eventStateRegistry.updateWorkerRoute.getState(execution)).toBe("updated");
  });
});
