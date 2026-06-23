import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import type { ConsolePanel } from "@/hooks/useCanvasData";
import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import type { ConsoleTriggerOptions } from "./ConsoleContext";
import { NodesPanelCard } from "./NodesPanelCard";

const NODE: SuperplaneComponentsNode = {
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
  id: "key-nodes",
  type: "nodes",
  content: {
    title: "Key nodes",
    nodes: [
      {
        node: "deploy-prod",
        description: "Deploys production",
        showRun: true,
        triggerName: "manual",
      },
    ],
  },
};

function renderPanel({
  canRunNodes,
  onTriggerNode,
  nodes = [NODE],
  panel = PANEL,
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: ConsoleTriggerOptions) => void;
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
      <NodesPanelCard panel={panel} readOnly onDelete={() => undefined} onChange={() => undefined} />
    </ConsoleContextProvider>,
  );
}

describe("NodesPanelCard run flow", () => {
  it("opens the confirm dialog instead of triggering immediately", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("nodes-panel-row-run"));
    expect(onTrigger).not.toHaveBeenCalled();
    expect(screen.getByTestId("nodes-panel-row-run-dialog-submit")).toBeTruthy();
  });

  it("submits the merged parameters via ctx.onTriggerNode on confirm", async () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("nodes-panel-row-run"));
    const branch = screen.getByLabelText("branch") as HTMLInputElement;
    fireEvent.change(branch, { target: { value: "release/v3" } });
    await act(async () => {
      fireEvent.click(screen.getByTestId("nodes-panel-row-run-dialog-submit"));
    });
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(1));
    expect(onTrigger).toHaveBeenCalledWith("node-1", {
      hookName: "run",
      templateName: "manual",
      parameters: { template: "manual", branch: "release/v3" },
    });
  });

  it("disables the Run button when the viewer cannot run nodes", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: false, onTriggerNode: onTrigger });
    const runButton = screen.getByTestId("nodes-panel-row-run");
    expect(runButton).toBeDisabled();
    fireEvent.click(runButton);
    expect(onTrigger).not.toHaveBeenCalled();
  });

  it("does not render the Run button for non-trigger nodes even when showRun is set", () => {
    renderPanel({
      canRunNodes: true,
      nodes: [NODE_ACTION],
      panel: {
        id: "key-nodes",
        type: "nodes",
        content: {
          title: "Key nodes",
          nodes: [{ node: "publish-artifact", showRun: true }],
        },
      },
    });
    expect(screen.queryByTestId("nodes-panel-row-run")).toBeNull();
  });
});
