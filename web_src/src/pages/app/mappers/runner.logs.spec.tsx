import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "bun:test";
import { ComponentBase } from "@/ui/componentBase";
import type { RunnerLiveLogDialogProps } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";
import { runnerMapper } from "./runner";
import type { ComponentBaseContext, ExecutionInfo } from "./types";

vi.mock("@/pages/factories/pages/work-order-split-run/PhaseLogCard", () => ({
  PhaseLogCard: () => <div data-testid="phase-log-card">logs</div>,
}));

function makeExecution(): ExecutionInfo {
  const now = new Date().toISOString();
  return {
    id: "execution-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    resultReason: "RESULT_REASON_ERROR",
    resultMessage: "Storybook deploy failed",
    metadata: {},
    configuration: {},
    rootEvent: {
      id: "event-1",
      createdAt: now,
      customName: "Pull request",
      data: {},
      nodeId: "trigger-1",
      type: "trigger",
    },
  };
}

function makeContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [
      {
        id: "trigger-1",
        name: "On Pull Request",
        componentName: "github.onPullRequest",
        isCollapsed: false,
        configuration: {},
        metadata: {},
      },
    ],
    node: {
      id: "node-runner-1",
      name: "Build & Deploy Storybook",
      componentName: "runnerBash",
      isCollapsed: false,
      configuration: {},
      metadata: {},
    },
    componentDefinition: {
      name: "runnerBash",
      label: "Run Shell Command",
      description: "Runs a Bash script on a fleet runner",
      icon: "terminal",
      color: "blue",
    },
    lastExecutions: [makeExecution()],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    canvasMode: "live",
    organizationId: "org-1",
    canvasId: "canvas-1",
    ...overrides,
  };
}

describe("runner factory logs", () => {
  it("passes canvas session ids so the logs dialog can fetch", () => {
    const props = runnerMapper.props(makeContext());
    const field = props.customField as React.ReactElement<RunnerLiveLogDialogProps>;

    expect(field.props.session).toEqual({
      organizationId: "org-1",
      canvasId: "canvas-1",
    });
    expect(field.props.component).toBe("runnerBash");
  });

  it("shows See logs on the factory runner card when a run exists", () => {
    const props = runnerMapper.props(makeContext());

    render(
      <ComponentBase
        {...props}
        isFactoryApp
        canvasMode="live"
        componentLabel="Run Shell Command"
        nodeName="Build & Deploy Storybook"
      />,
    );

    expect(screen.getByRole("button", { name: "See logs" })).toBeInTheDocument();
  });

  it("hides See logs when the runner has not executed", () => {
    const props = runnerMapper.props(makeContext({ lastExecutions: [] }));

    render(
      <ComponentBase
        {...props}
        isFactoryApp
        canvasMode="live"
        componentLabel="Run Shell Command"
        nodeName="Build & Deploy Storybook"
      />,
    );

    expect(screen.queryByRole("button", { name: "See logs" })).not.toBeInTheDocument();
  });
});
