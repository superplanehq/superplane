import { describe, expect, it, vi } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { baseMapper } from "./base";

vi.mock("..", () => ({
  getState: () => () => "completed",
  getStateMap: () => ({}),
  getTriggerRenderer: () => ({
    getTitleAndSubtitle: () => ({ title: "Manual run", subtitle: "" }),
  }),
}));

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "OpenAI Prompt",
    componentName: "openai.textPrompt",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "openai.response",
    timestamp: "2026-01-02T03:04:05.000Z",
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-01-02T03:00:00.000Z",
    updatedAt: "2026-01-02T03:04:05.000Z",
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

const definition: ComponentDefinition = {
  name: "openai.textPrompt",
  label: "Text Prompt",
  description: "",
  icon: "sparkles",
  color: "gray",
};

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: definition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

function buildDetailsContext(overrides?: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node = buildNode();
  return {
    nodes: [node],
    node,
    execution: buildExecution(overrides),
  };
}

describe("openai baseMapper.props", () => {
  it("shows configured model metadata", () => {
    const props = baseMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: "gpt-5.2" } }),
      }),
    );

    expect(props.metadata).toEqual([{ icon: "cpu", label: "gpt-5.2" }]);
  });

  it("omits model metadata when not configured", () => {
    const props = baseMapper.props(buildPropsContext());

    expect(props.metadata).toEqual([]);
  });
});

describe("openai baseMapper.getExecutionDetails", () => {
  it("includes model and token usage from the response payload", () => {
    const details = baseMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            buildOutput({
              model: "gpt-5.2",
              usage: {
                input_tokens: 1200,
                output_tokens: 345,
                total_tokens: 1545,
              },
            }),
          ],
        },
      }),
    );

    expect(details["Started At"]).toBeDefined();
    expect(details["Event Type"]).toBe("openai.response");
    expect(details["Model"]).toBe("gpt-5.2");
    expect(details["Tokens"]).toBe("1,545 (1,200 in / 345 out)");
    expect(details["Emitted At"]).toBeDefined();
  });

  it("falls back to zero when input and output token counts are missing", () => {
    const details = baseMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [buildOutput({ usage: { total_tokens: 100 } })],
        },
      }),
    );

    expect(details["Tokens"]).toBe("100 (0 in / 0 out)");
  });

  it("returns Started At and no payload-derived fields when outputs are absent", () => {
    const details = baseMapper.getExecutionDetails(buildDetailsContext({ outputs: undefined }));

    expect(details["Started At"]).toBeDefined();
    expect(details["Event Type"]).toBeUndefined();
    expect(details["Model"]).toBeUndefined();
    expect(details["Tokens"]).toBeUndefined();
    expect(details["Emitted At"]).toBeUndefined();
  });
});
