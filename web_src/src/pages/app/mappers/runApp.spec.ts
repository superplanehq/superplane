import { describe, expect, it } from "vitest";

import { runAppMapper, runAppStateFunction } from "./runApp";
import type { ComponentBaseContext, ComponentDefinition, ExecutionInfo, NodeInfo } from "./types";

const defaultDefinition: ComponentDefinition = {
  name: "runApp",
  label: "Run App",
  description: "",
  icon: "play",
  color: "gray",
};

function buildRunAppExecution(overrides: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: overrides.metadata ?? {},
    configuration: {},
    ...overrides,
  } as ExecutionInfo;
}

function buildContext(node: NodeInfo): ComponentBaseContext {
  return {
    nodes: [node],
    node,
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("runAppStateFunction", () => {
  it("shows failed when the child run timed out", () => {
    expect(
      runAppStateFunction(
        buildRunAppExecution({
          metadata: {
            run: { id: "child-run", result: "cancelled", error: "timed out after 2s" },
          },
        }),
      ),
    ).toBe("failed");
  });

  it("shows failed when the child run was cancelled", () => {
    expect(
      runAppStateFunction(
        buildRunAppExecution({
          metadata: {
            run: { id: "child-run", result: "cancelled" },
          },
        }),
      ),
    ).toBe("failed");
  });

  it("shows success when the child run passed", () => {
    expect(
      runAppStateFunction(
        buildRunAppExecution({
          metadata: {
            run: { id: "child-run", result: "passed" },
          },
        }),
      ),
    ).toBe("success");
  });
});

describe("runAppMapper.props metadata", () => {
  it("omits timeout metadata when timeout is not configured", () => {
    const node: NodeInfo = {
      id: "run-app",
      name: "Run Child",
      componentName: "runApp",
      isCollapsed: false,
      configuration: {
        app: "child-app-id",
        node: "on-run",
        parameters: {},
      },
      metadata: {
        app: { id: "child-app-id", name: "Child App" },
        node: { id: "on-run", name: "On Run" },
      },
    };

    const props = runAppMapper.props(buildContext(node));

    expect(props.metadata).toEqual([
      { icon: "layout-grid", label: "Child App" },
      { icon: "workflow", label: "On Run" },
    ]);
  });

  it("includes configured timeout in metadata", () => {
    const node: NodeInfo = {
      id: "run-app",
      name: "Run Child",
      componentName: "runApp",
      isCollapsed: false,
      configuration: {
        app: "child-app-id",
        node: "on-run",
        parameters: {},
        timeout: 3600,
      },
      metadata: {
        app: { id: "child-app-id", name: "Child App" },
        node: { id: "on-run", name: "On Run" },
      },
    };

    const props = runAppMapper.props(buildContext(node));

    expect(props.metadata).toEqual([
      { icon: "layout-grid", label: "Child App" },
      { icon: "workflow", label: "On Run" },
      { icon: "clock", label: "Timeout: 1h" },
    ]);
  });

  it("formats non-default configured timeout values", () => {
    const node: NodeInfo = {
      id: "run-app",
      name: "Run Child",
      componentName: "runApp",
      isCollapsed: false,
      configuration: {
        app: "child-app-id",
        node: "on-run",
        parameters: {},
        timeout: 120,
      },
      metadata: {
        app: { id: "child-app-id", name: "Child App" },
        node: { id: "on-run", name: "On Run" },
      },
    };

    const props = runAppMapper.props(buildContext(node));

    expect(props.metadata?.at(-1)).toEqual({ icon: "clock", label: "Timeout: 2m" });
  });
});
