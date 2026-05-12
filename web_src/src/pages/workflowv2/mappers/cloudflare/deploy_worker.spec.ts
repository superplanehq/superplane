import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
} from "../types";
import { deployWorkerMapper } from "./deploy_worker";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.deployWorker",
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

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.deployWorker",
  label: "Deploy Worker",
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

describe("deployWorkerMapper.getExecutionDetails", () => {
  it("includes script only when data is present", () => {
    const data = {
      scriptName: "my-worker",
      versionId: "ver-1",
      deployment: { id: "dep-1" },
    };
    const outputs = { default: [{ type: "cloudflare.worker.deployed", timestamp: new Date().toISOString(), data }] };
    const details = deployWorkerMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs } }));
    expect(details["Script"]).toBe("my-worker");
    expect(details["Version ID"]).toBeUndefined();
    expect(details["Deployment ID"]).toBeUndefined();
  });
});

describe("deployWorkerMapper.props", () => {
  it("shows at most three chips: name, source, provision", () => {
    const props = deployWorkerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { scriptName: "w", source: "inline" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "code", label: "w" },
      { icon: "file-text", label: "Inline" },
      { icon: "package", label: "Provision on" },
    ]);
  });

  it("shows provision off when disabled", () => {
    const props = deployWorkerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { scriptName: "w", source: "inline", provisionIfMissing: false },
        }),
      }),
    );
    expect(props.metadata?.[2]).toEqual({ icon: "package", label: "Provision off" });
  });
});

describe("eventStateRegistry.deployWorker", () => {
  it("maps deploy output to deployed", () => {
    const execution = buildExecution({
      outputs: {
        default: [
          {
            type: "cloudflare.worker.deployed",
            timestamp: new Date().toISOString(),
            data: {},
          },
        ],
      },
    });

    expect(eventStateRegistry.deployWorker.getState(execution)).toBe("deployed");
  });
});
