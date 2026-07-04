import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { createDeploymentMapper } from "./create_deployment";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Deployment",
    componentName: "github.createDeployment",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "github.deployment",
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
  name: "github.createDeployment",
  label: "Create Deployment",
  description: "",
  icon: "github",
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

describe("createDeploymentMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createDeploymentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createDeploymentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("shows deployment URL and omits id and sha in details", () => {
    const apiUrl = "https://api.github.com/repos/o/r/deployments/99";
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              id: 99,
              sha: "abc123",
              ref: "feature/x",
              environment: "preview-pr-1",
              description: "test",
              url: apiUrl,
              created_at: "2026-01-16T12:00:00Z",
              updated_at: "2026-01-16T12:01:00Z",
            }),
          ],
        },
      },
    });
    const details = createDeploymentMapper.getExecutionDetails(ctx);
    expect(details["Deployment URL"]).toBe(apiUrl);
    expect(details["Environment"]).toBe("preview-pr-1");
    expect(details["Ref"]).toBe("feature/x");
    expect(details["Description"]).toBe("test");
    expect(details["Deployment ID"]).toBeUndefined();
    expect(details["SHA"]).toBeUndefined();
    expect(details["Updated At"]).toBeUndefined();
  });

  it("does not include Updated At when present on payload", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              url: "https://api.github.com/repos/o/r/deployments/1",
              updated_at: "2026-01-16T12:01:00Z",
            }),
          ],
        },
      },
    });
    const details = createDeploymentMapper.getExecutionDetails(ctx);
    expect(details["Updated At"]).toBeUndefined();
  });
});

describe("createDeploymentMapper.props", () => {
  it("does not throw with minimal context", () => {
    const ctx = buildPropsContext();
    expect(() => createDeploymentMapper.props!(ctx)).not.toThrow();
  });
});
