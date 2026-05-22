import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { deleteTagMapper } from "./delete_tag";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Tag",
    componentName: "dockerhub.deleteTag",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "dockerhub.deletedTag",
    timestamp: new Date().toISOString(),
    data,
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

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "dockerhub.deleteTag",
  label: "Delete Tag",
  description: "",
  icon: "docker",
  color: "gray",
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

describe("deleteTagMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteTagMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteTagMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns empty object when result is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(deleteTagMapper.getExecutionDetails(ctx)).toEqual({});
  });

  it("returns namespace, repository and tag from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              namespace: "superplane",
              repository: "demo",
              tag: "v1.2.3-rc1",
            }),
          ],
        },
      },
    });
    const details = deleteTagMapper.getExecutionDetails(ctx);
    expect(details["Namespace"]).toBe("superplane");
    expect(details["Repository"]).toBe("demo");
    expect(details["Tag"]).toBe("v1.2.3-rc1");
  });

  it("returns dashes for missing fields", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({})],
        },
      },
    });
    const details = deleteTagMapper.getExecutionDetails(ctx);
    expect(details["Namespace"]).toBe("-");
    expect(details["Repository"]).toBe("-");
    expect(details["Tag"]).toBe("-");
  });
});

describe("deleteTagMapper.props", () => {
  it("does not throw with minimal context", () => {
    const ctx = buildPropsContext();
    expect(() => deleteTagMapper.props!(ctx)).not.toThrow();
  });

  it("includes repository and tag in metadata when configured", () => {
    const ctx = buildPropsContext({
      node: buildNode({
        configuration: { repository: "superplane/demo", tag: "v1.2.3-rc1" },
      }),
    });
    const props = deleteTagMapper.props!(ctx);
    expect(props.metadata?.some((m) => m.label === "superplane/demo")).toBe(true);
    expect(props.metadata?.some((m) => m.label === "v1.2.3-rc1")).toBe(true);
  });
});
