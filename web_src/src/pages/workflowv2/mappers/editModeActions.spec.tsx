import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { ComponentBase } from "@/ui/componentBase";
import { Trigger } from "@/ui/trigger";
import { approvalMapper } from "./approval";
import { startTriggerRenderer } from "./start";
import { timeGateMapper } from "./timegate";
import { waitMapper } from "./wait";
import type { ComponentBaseContext, ExecutionInfo, TriggerRendererContext } from "./types";

function makeExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  const now = new Date().toISOString();

  return {
    id: "execution-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_PENDING",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: {
      id: "event-1",
      createdAt: now,
      customName: "Start event",
      data: {},
      nodeId: "trigger-1",
      type: "trigger",
    },
    ...overrides,
  };
}

function makeComponentBaseContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [
      {
        id: "trigger-1",
        name: "Start",
        componentName: "start",
        isCollapsed: false,
        configuration: {},
        metadata: {},
      },
    ],
    node: {
      id: "node-1",
      name: "Node",
      componentName: "test",
      isCollapsed: false,
      configuration: {},
      metadata: {},
    },
    componentDefinition: {
      name: "test",
      label: "Test",
      description: "",
      icon: "play",
      color: "orange",
    },
    lastExecutions: [],
    currentUser: {
      id: "user-1",
      name: "User",
      email: "user@example.com",
      roles: [],
      groups: [],
    },
    actions: {
      invokeNodeExecutionHook: vi.fn(),
    },
    ...overrides,
  };
}

function makeTriggerContext(overrides?: Partial<TriggerRendererContext>): TriggerRendererContext {
  return {
    node: {
      id: "trigger-1",
      name: "Start",
      componentName: "start",
      isCollapsed: false,
      configuration: {
        templates: [{ name: "Example", payload: { ok: true } }],
      },
    },
    definition: {
      name: "start",
      label: "Start",
      description: "",
      icon: "play",
      color: "purple",
    },
    lastEvent: undefined,
    ...overrides,
  };
}

describe("workflow v2 edit-mode action affordances", () => {
  it("hides start trigger run buttons in edit mode", () => {
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "edit",
    });

    render(<Trigger {...props} canvasMode="edit" onRun={vi.fn()} />);

    expect(screen.queryByTestId("start-template-run")).not.toBeInTheDocument();
    expect(screen.getByText("Example")).toBeInTheDocument();
  });

  it("keeps start trigger run buttons in live mode", () => {
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "live",
    });

    render(<Trigger {...props} canvasMode="live" onRun={vi.fn()} />);

    expect(screen.getByTestId("start-template-run")).toBeInTheDocument();
  });

  it("disables approval actions in edit mode", () => {
    const props = approvalMapper.props(
      makeComponentBaseContext({
        canvasMode: "edit",
        componentDefinition: {
          name: "approval",
          label: "Approval",
          description: "",
          icon: "hand",
          color: "orange",
        },
        lastExecutions: [
          makeExecution({
            metadata: {
              records: [
                {
                  index: 0,
                  state: "pending",
                  type: "user",
                  user: {
                    id: "user-1",
                    name: "User",
                    email: "user@example.com",
                    roles: [],
                    groups: [],
                  },
                },
              ],
            },
          }),
        ],
      }),
    );

    expect(React.isValidElement(props.customField)).toBe(true);
    const approvalField = props.customField as React.ReactElement<{
      approvals: Array<{ interactive: boolean }>;
    }>;

    expect(approvalField.props.approvals[0].interactive).toBe(false);
  });

  it("hides wait and time gate push-through actions in edit mode", () => {
    const waitProps = waitMapper.props(
      makeComponentBaseContext({
        canvasMode: "edit",
        componentDefinition: {
          name: "wait",
          label: "Wait",
          description: "",
          icon: "clock",
          color: "orange",
        },
        lastExecutions: [makeExecution()],
      }),
    );
    const timeGateProps = timeGateMapper.props(
      makeComponentBaseContext({
        canvasMode: "edit",
        componentDefinition: {
          name: "timeGate",
          label: "Time Gate",
          description: "",
          icon: "clock",
          color: "orange",
        },
        lastExecutions: [makeExecution()],
      }),
    );

    render(
      <>
        <ComponentBase {...waitProps} canvasMode="edit" />
        <ComponentBase {...timeGateProps} canvasMode="edit" />
      </>,
    );

    expect(waitProps.customFieldVisibility).toBe("live-only");
    expect(timeGateProps.customFieldVisibility).toBe("live-only");
    expect(screen.queryByText("Push through")).not.toBeInTheDocument();
  });
});
