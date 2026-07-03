import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import type { ConsolePanel } from "@/hooks/useCanvasData";
import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import type { ConsoleTriggerOptions } from "./ConsoleContext";
import { NodePanelCard } from "./NodePanelCard";

const NODE_NO_PARAMS: SuperplaneComponentsNode = {
  id: "node-1",
  name: "deploy-prod",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "manual", payload: { reason: "console" } }],
  },
};

const NODE_WITH_PARAMS: SuperplaneComponentsNode = {
  id: "node-1",
  name: "deploy-prod",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [
      {
        name: "manual",
        payload: { reason: "console" },
        parameters: [{ name: "branch", type: "string", defaultString: "main" }],
      },
    ],
  },
};

const NODE_ACTION: SuperplaneComponentsNode = {
  id: "node-2",
  name: "publish-artifact",
  type: "TYPE_ACTION",
};

const PANEL: ConsolePanel = {
  id: "deploy",
  type: "node",
  content: {
    title: "Deploy",
    node: "deploy-prod",
    showRun: true,
    triggerName: "manual",
  },
};

const PANEL_PROMPT: ConsolePanel = {
  id: "deploy",
  type: "node",
  content: {
    title: "Deploy",
    node: "deploy-prod",
    showRun: true,
    triggerName: "manual",
    promptConfirmation: true,
  },
};

function renderPanel({
  canRunNodes,
  onTriggerNode,
  nodes = [NODE_NO_PARAMS],
  panel = PANEL,
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: ConsoleTriggerOptions) => void | Promise<void>;
  nodes?: SuperplaneComponentsNode[];
  panel?: ConsolePanel;
}) {
  return render(
    <ConsoleContextProvider
      canvasId="canvas-1"
      organizationId="org-1"
      nodes={nodes}
      canRunNodes={canRunNodes}
      onTriggerNode={onTriggerNode}
    >
      <NodePanelCard panel={panel} readOnly onDelete={() => undefined} onChange={() => undefined} />
    </ConsoleContextProvider>,
  );
}

describe("NodePanelCard run flow", () => {
  it("triggers immediately for a parameter-less template when confirmation is not required", async () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    await act(async () => {
      fireEvent.click(screen.getByTestId("node-panel-run"));
    });
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(1));
    expect(onTrigger).toHaveBeenCalledWith("node-1", {
      hookName: "run",
      templateName: "manual",
      parameters: { template: "manual" },
    });
    expect(screen.queryByTestId("node-panel-run-dialog-submit")).toBeNull();
  });

  it("locks the Run button with a loading state while the direct trigger is in flight", async () => {
    let resolveTrigger: (() => void) | undefined;
    const onTrigger = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveTrigger = resolve;
        }),
    );
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    await waitFor(() => expect(screen.getByTestId("node-panel-run")).toBeDisabled());
    await act(async () => {
      resolveTrigger?.();
    });
    await waitFor(() => expect(screen.getByTestId("node-panel-run")).not.toBeDisabled());
  });

  it("opens the confirm dialog for a parameter-less template when promptConfirmation is enabled", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, panel: PANEL_PROMPT });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    expect(onTrigger).not.toHaveBeenCalled();
    expect(screen.getByTestId("node-panel-run-dialog-submit")).toBeTruthy();
  });

  it("opens the confirm dialog for templates with input fields even without promptConfirmation", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, nodes: [NODE_WITH_PARAMS] });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    expect(onTrigger).not.toHaveBeenCalled();
    expect(screen.getByTestId("node-panel-run-dialog-submit")).toBeTruthy();
    expect(screen.getByLabelText("branch")).toBeTruthy();
  });

  it("submits the preview parameters via ctx.onTriggerNode on confirm", async () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, panel: PANEL_PROMPT });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    await act(async () => {
      fireEvent.click(screen.getByTestId("node-panel-run-dialog-submit"));
    });
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(1));
    expect(onTrigger).toHaveBeenCalledWith("node-1", {
      hookName: "run",
      templateName: "manual",
      parameters: { template: "manual" },
    });
  });

  it("closes the dialog on confirm and drives the loading state on the widget button", async () => {
    let resolveTrigger: (() => void) | undefined;
    const onTrigger = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveTrigger = resolve;
        }),
    );
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, panel: PANEL_PROMPT });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    fireEvent.click(screen.getByTestId("node-panel-run-dialog-submit"));
    await waitFor(() => expect(screen.queryByTestId("node-panel-run-dialog-submit")).toBeNull());
    expect(screen.getByTestId("node-panel-run")).toBeDisabled();
    await act(async () => {
      resolveTrigger?.();
    });
    await waitFor(() => expect(screen.getByTestId("node-panel-run")).not.toBeDisabled());
  });

  it("disables the Run button when the viewer cannot run nodes", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: false, onTriggerNode: onTrigger });
    const runButton = screen.getByTestId("node-panel-run");
    expect(runButton).toBeDisabled();
    fireEvent.click(runButton);
    expect(onTrigger).not.toHaveBeenCalled();
  });

  it("renders the custom label override instead of the resolved node name", () => {
    renderPanel({
      canRunNodes: true,
      panel: {
        id: "deploy",
        type: "node",
        content: {
          title: "Deploy",
          node: "deploy-prod",
          label: "Ship to prod",
          showRun: false,
        },
      },
    });
    expect(screen.getByTestId("node-panel-name").textContent).toBe("Ship to prod");
  });

  it("does not render the Run button for non-trigger nodes even when showRun is set", () => {
    renderPanel({
      canRunNodes: true,
      nodes: [NODE_ACTION],
      panel: {
        id: "publish",
        type: "node",
        content: { title: "Publish", node: "publish-artifact", showRun: true },
      },
    });
    expect(screen.queryByTestId("node-panel-run")).toBeNull();
  });
});
