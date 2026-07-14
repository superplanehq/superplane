import { describe, expect, it } from "vitest";

import { createBatchMessageMapper } from "./create_batch_message";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

const defaultDefinition: ComponentDefinition = {
  name: "claude.createBatchMessage",
  label: "Create Batch Message",
  description: "",
  icon: "layers",
  color: "#D97757",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.createBatchMessage",
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
  return { type: "claude.createBatchMessage.result", timestamp: new Date().toISOString(), data };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

describe("createBatchMessageMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    expect(() =>
      createBatchMessageMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } })),
    ).not.toThrow();
    expect(() =>
      createBatchMessageMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("surfaces executed-at first, then status/batchId/progress/succeeded/errored from output data", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              status: "ended",
              batchId: "msgbatch_1",
              requestCounts: { processing: 0, succeeded: 3, errored: 1, canceled: 0, expired: 0 },
            }),
          ],
        },
      },
    });
    const details = createBatchMessageMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Status"]).toBe("ended");
    expect(details["Batch ID"]).toBe("msgbatch_1");
    expect(details["Progress"]).toBe("4 / 4 complete");
    expect(details["Succeeded"]).toBe("3");
    expect(details["Errored"]).toBe("1");
  });

  it("falls back to execution metadata when there is no output yet (batch still running)", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: undefined,
        metadata: {
          status: "in_progress",
          batchId: "msgbatch_2",
          requestCounts: { processing: 5, succeeded: 2, errored: 0, canceled: 0, expired: 0 },
        },
      },
    });
    const details = createBatchMessageMapper.getExecutionDetails(ctx);
    expect(details["Status"]).toBe("in_progress");
    expect(details["Batch ID"]).toBe("msgbatch_2");
    expect(details["Progress"]).toBe("2 / 7 complete");
    expect(details["Succeeded"]).toBe("2");
    expect(details["Errored"]).toBe("0");
  });

  it("omits Progress when there are no request counts anywhere", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({ status: "ended" })] } } });
    const details = createBatchMessageMapper.getExecutionDetails(ctx);
    expect(details["Progress"]).toBeUndefined();
    expect(details["Succeeded"]).toBeUndefined();
    expect(details["Errored"]).toBeUndefined();
  });
});

describe("createBatchMessageMapper.props", () => {
  it("shows the model and 'Single prompt' by default", () => {
    const props = createBatchMessageMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { model: "claude-opus-4-6" } }) }),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-opus-4-6" },
      { icon: "layers", label: "Single prompt" },
    ]);
  });

  it("renders the model name when the model is an integration-resource object, not the raw object", () => {
    const props = createBatchMessageMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: { id: "m_1", name: "claude-opus-4-6", type: "model" } } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-opus-4-6" },
      { icon: "layers", label: "Single prompt" },
    ]);
  });

  it("shows 'Multiple prompts' when mode is multiple", () => {
    const props = createBatchMessageMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: "claude-opus-4-6", mode: "multiple" } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-opus-4-6" },
      { icon: "layers", label: "Multiple prompts" },
    ]);
  });

  it("appends progress from the last execution's output once a batch has run", () => {
    const lastExecution = buildExecution({
      outputs: {
        default: [buildOutput({ requestCounts: { processing: 1, succeeded: 2, errored: 0, canceled: 0, expired: 0 } })],
      },
      rootEvent: {
        id: "event-1",
        createdAt: new Date().toISOString(),
        data: {},
        nodeId: "trigger-1",
        type: "manual.run",
      },
    });
    const props = createBatchMessageMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: "claude-opus-4-6" } }),
        lastExecutions: [lastExecution],
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-opus-4-6" },
      { icon: "layers", label: "Single prompt" },
      { icon: "check-circle", label: "2 / 3 complete" },
    ]);
  });

  it("marks includeEmptyState when there is no last execution, and not when there is one", () => {
    expect(createBatchMessageMapper.props(buildPropsContext({ lastExecutions: [] })).includeEmptyState).toBe(true);
    const lastExecution = buildExecution({
      rootEvent: {
        id: "event-1",
        createdAt: new Date().toISOString(),
        data: {},
        nodeId: "trigger-1",
        type: "manual.run",
      },
    });
    expect(
      createBatchMessageMapper.props(buildPropsContext({ lastExecutions: [lastExecution] })).includeEmptyState,
    ).toBe(false);
  });
});

describe("eventStateRegistry.createBatchMessage", () => {
  it("maps finished passed to completed", () => {
    expect(eventStateRegistry.createBatchMessage.getState(buildExecution())).toBe("completed");
  });
});
