import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";

const definition: ComponentDefinition = {
  name: "groq.chatCompletion",
  label: "Text prompt",
  description: "",
  icon: "message-square",
  color: "orange",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "groq.chatCompletion",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildContext(node: NodeInfo): ComponentBaseContext {
  return {
    nodes: [],
    node,
    componentDefinition: definition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("groq baseMapper", () => {
  it("shows the selected model on the node", () => {
    const props = baseMapper.props(buildContext(buildNode({ configuration: { model: "llama-3.3-70b-versatile" } })));

    expect(props.metadata).toEqual([{ icon: "sparkles", label: "llama-3.3-70b-versatile" }]);
    expect(props.iconSrc).toBeTruthy();
  });

  it("shows model and token details for a completion", () => {
    const details = baseMapper.getExecutionDetails!({
      nodes: [buildNode()],
      node: buildNode(),
      execution: {
        id: "execution-1",
        createdAt: "2026-08-25T10:00:00.000Z",
        updatedAt: "2026-08-25T10:00:00.000Z",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        resultReason: "RESULT_REASON_OK",
        resultMessage: "",
        metadata: {},
        configuration: {},
        rootEvent: undefined,
        outputs: {
          default: [
            {
              type: "groq.chatCompletion.result",
              timestamp: "2026-08-25T10:00:00.000Z",
              data: {
                model: "llama-3.3-70b-versatile",
                usage: { prompt_tokens: 4, completion_tokens: 3, total_tokens: 7 },
              },
            },
          ],
        },
      },
    });

    expect(details.Model).toBe("llama-3.3-70b-versatile");
    expect(details["Tokens"]).toBe("7 (4 in / 3 out)");
  });
});
