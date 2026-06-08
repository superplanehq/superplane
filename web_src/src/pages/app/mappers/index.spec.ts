import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { makeComponentsNode } from "@/test/factories";
import { getComponentBaseMapper, getExecutionDetails, getStateMap } from "./index";
import { RUNNER_STATE_REGISTRY } from "./runner";

function makeNode(name: string): ComponentsNode {
  return makeComponentsNode({
    id: "node-1",
    component: name,
  });
}

function makeExecution(): CanvasesCanvasNodeExecution {
  return {
    id: "execution-1",
    createdAt: new Date().toISOString(),
  } as CanvasesCanvasNodeExecution;
}

describe("getExecutionDetails", () => {
  it("returns undefined when no top-level mapper is registered", () => {
    expect(getExecutionDetails("unknown-component", makeExecution(), makeNode("unknown-component"))).toBeUndefined();
  });

  it("returns undefined when no app mapper is registered", () => {
    expect(getExecutionDetails("unknown.component", makeExecution(), makeNode("unknown.component"))).toBeUndefined();
  });

  it("returns undefined when an app exists but the component mapper does not", () => {
    expect(getExecutionDetails("github.unknown", makeExecution(), makeNode("github.unknown"))).toBeUndefined();
  });

  it("resolves runnerBash mapper and state registry", () => {
    const mapper = getComponentBaseMapper("runnerBash");
    const props = mapper.props({
      node: {
        id: "node-runbash-1",
        name: "Run Bash",
        componentName: "runnerBash",
        isCollapsed: false,
        configuration: {
          machine_type: "aws-standard-1",
          script: 'echo \'{"ok":true}\' > "$SUPERPLANE_RESULT_FILE"',
        },
        metadata: {},
      },
      nodes: [],
      componentDefinition: {
        name: "runnerBash",
        label: "Run Bash",
        description: "Runs a Bash script on a fleet runner with upstream node data in SUPERPLANE_PAYLOAD_FILE",
        icon: "code",
        color: "blue",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
      canvasMode: "live",
    });

    expect(props.customField).toBeDefined();
    expect(getStateMap("runnerBash")).toBe(RUNNER_STATE_REGISTRY.stateMap);
  });

  it("resolves runnerJS mapper and state registry", () => {
    const mapper = getComponentBaseMapper("runnerJS");
    const props = mapper.props({
      node: {
        id: "node-runjs-1",
        name: "Run JavaScript",
        componentName: "runnerJS",
        isCollapsed: false,
        configuration: {
          machine_type: "aws-standard-1",
          script: "function main() { return { ok: true }; }",
        },
        metadata: {},
      },
      nodes: [],
      componentDefinition: {
        name: "runnerJS",
        label: "Run JavaScript",
        description: "Runs JavaScript on a fleet runner",
        icon: "code",
        color: "blue",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
      canvasMode: "live",
    });

    expect(props.customField).toBeDefined();
    expect(getStateMap("runnerJS")).toBe(RUNNER_STATE_REGISTRY.stateMap);
  });

  it("resolves runnerPython mapper and state registry", () => {
    const mapper = getComponentBaseMapper("runnerPython");
    const props = mapper.props({
      node: {
        id: "node-runpy-1",
        name: "Run Python",
        componentName: "runnerPython",
        isCollapsed: false,
        configuration: {
          machine_type: "aws-standard-1",
          script: 'def main(payload):\n    return {"ok": True}',
        },
        metadata: {},
      },
      nodes: [],
      componentDefinition: {
        name: "runnerPython",
        label: "Run Python",
        description: "Runs Python on a fleet runner",
        icon: "code",
        color: "blue",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
      canvasMode: "live",
    });

    expect(props.customField).toBeDefined();
    expect(getStateMap("runnerPython")).toBe(RUNNER_STATE_REGISTRY.stateMap);
  });
});
