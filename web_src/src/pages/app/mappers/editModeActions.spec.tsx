import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

    render(<Trigger {...props} canvasMode="edit" />);

    expect(screen.queryByTestId("start-template-run")).not.toBeInTheDocument();
    expect(screen.getByText("Example")).toBeInTheDocument();
  });

  it("keeps start trigger run buttons in live mode", () => {
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook: vi.fn().mockResolvedValue(undefined), openModal: vi.fn() },
    });

    render(<Trigger {...props} canvasMode="live" />);

    expect(screen.getByTestId("start-template-run")).toBeInTheDocument();
  });

  it("hides start trigger run buttons in live mode when actions are unavailable", () => {
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "live",
      actions: undefined,
    });

    render(<Trigger {...props} canvasMode="live" />);

    expect(screen.queryByTestId("start-template-run")).not.toBeInTheDocument();
    expect(screen.getByText("Example")).toBeInTheDocument();
  });

  it("opens a parameter modal for manual run templates with parameters", async () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext({
        node: {
          id: "trigger-1",
          name: "Start",
          componentName: "start",
          isCollapsed: false,
          configuration: {
            templates: [
              {
                name: "Example",
                payload: { message: "default" },
                parameters: [{ name: "message", type: "string", defaultString: "default" }],
              },
            ],
          },
        },
      }),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    fireEvent.click(screen.getByTestId("start-template-run"));

    expect(openModal).toHaveBeenCalledTimes(1);
    expect(openModal).toHaveBeenCalledWith(expect.objectContaining({ title: "Start" }));
    expect(openModal.mock.calls[0][0]).not.toHaveProperty("description");
    expect(invokeNodeTriggerHook).not.toHaveBeenCalled();

    const modal = openModal.mock.calls[0][0] as {
      content: (ctx: { close: () => void }) => React.ReactNode;
    };
    render(<>{modal.content({ close: vi.fn() })}</>);

    fireEvent.change(screen.getByLabelText("message"), { target: { value: "from form" } });
    fireEvent.click(screen.getByTestId("emit-event-submit-button"));

    await waitFor(() =>
      expect(invokeNodeTriggerHook).toHaveBeenCalledWith("run", {
        template: "Example",
        message: "from form",
      }),
    );
  });

  it("submits select parameter values from the manual run modal", async () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext({
        node: {
          id: "trigger-1",
          name: "Start",
          componentName: "start",
          isCollapsed: false,
          configuration: {
            templates: [
              {
                name: "Example",
                payload: { provider: '{{ parameters["provider"] }}' },
                parameters: [
                  {
                    name: "provider",
                    type: "select",
                    title: "LLM Provider",
                    defaultString: "openai",
                    options: [
                      { label: "OpenAI", value: "openai" },
                      { label: "Anthropic", value: "anthropic" },
                    ],
                  },
                ],
              },
            ],
          },
        },
      }),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    fireEvent.click(screen.getByTestId("start-template-run"));

    const modal = openModal.mock.calls[0][0] as {
      content: (ctx: { close: () => void }) => React.ReactNode;
    };
    render(<>{modal.content({ close: vi.fn() })}</>);

    expect(screen.getByText("LLM Provider")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("emit-event-submit-button"));

    await waitFor(() =>
      expect(invokeNodeTriggerHook).toHaveBeenCalledWith("run", {
        template: "Example",
        provider: "openai",
      }),
    );
  });

  it("invokes manual run directly when templates have no parameters", () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    fireEvent.click(screen.getByTestId("start-template-run"));

    expect(openModal).not.toHaveBeenCalled();
    expect(invokeNodeTriggerHook).toHaveBeenCalledWith("run", {
      template: "Example",
    });
  });

  it("invokes manual run directly when template parameters array is empty", () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext({
        node: {
          id: "trigger-1",
          name: "Start",
          componentName: "start",
          isCollapsed: false,
          configuration: {
            templates: [
              {
                name: "Example",
                payload: { ok: true },
                parameters: [],
              },
            ],
          },
        },
      }),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    fireEvent.click(screen.getByTestId("start-template-run"));

    expect(openModal).not.toHaveBeenCalled();
    expect(invokeNodeTriggerHook).toHaveBeenCalledWith("run", {
      template: "Example",
    });
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
