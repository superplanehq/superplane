import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { createDeploymentStatusMapper } from "./create_deployment_status";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Deployment Status",
    componentName: "github.createDeploymentStatus",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "github.deploymentStatus",
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
  name: "github.createDeploymentStatus",
  label: "Create Deployment Status",
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

describe("createDeploymentStatusMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createDeploymentStatusMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createDeploymentStatusMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("shows deployment status URL and omits id, environment url, log url", () => {
    const statusUrl = "https://api.github.com/repos/o/r/deployments/1/statuses/2";
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              id: 2,
              state: "success",
              environment: "preview-pr-1",
              description: "live",
              environment_url: "https://preview.example.com",
              log_url: "https://logs.example.com/run/1",
              url: statusUrl,
              created_at: "2026-01-16T12:00:00Z",
              updated_at: "2026-01-16T12:01:00Z",
            }),
          ],
        },
      },
    });
    const details = createDeploymentStatusMapper.getExecutionDetails(ctx);
    expect(details["Deployment Status URL"]).toBe(statusUrl);
    expect(details["State"]).toBe("success");
    expect(details["Environment"]).toBe("preview-pr-1");
    expect(details["Description"]).toBe("live");
    expect(details["Status ID"]).toBeUndefined();
    expect(details["Environment URL"]).toBeUndefined();
    expect(details["Log URL"]).toBeUndefined();
    expect(details["Updated At"]).toBeUndefined();
  });

  it("does not include Updated At when present on payload", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              url: "https://api.github.com/repos/o/r/deployments/1/statuses/2",
              updated_at: "2026-01-16T12:01:00Z",
            }),
          ],
        },
      },
    });
    const details = createDeploymentStatusMapper.getExecutionDetails(ctx);
    expect(details["Updated At"]).toBeUndefined();
  });
});

describe("createDeploymentStatusMapper.props", () => {
  it("does not throw with minimal context", () => {
    const ctx = buildPropsContext();
    expect(() => createDeploymentStatusMapper.props!(ctx)).not.toThrow();
  });
});
