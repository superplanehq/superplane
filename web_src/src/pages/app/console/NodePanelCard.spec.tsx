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
  it("opens the confirm dialog instead of triggering immediately", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    expect(onTrigger).not.toHaveBeenCalled();
    expect(screen.getByTestId("node-panel-run-dialog-submit")).toBeTruthy();
  });

  it("submits the preview parameters via ctx.onTriggerNode on confirm", async () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
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

  it("keeps the confirm dialog open when onTriggerNode rejects", async () => {
    const onTrigger = vi.fn().mockRejectedValue(new Error("API failed"));
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    await act(async () => {
      fireEvent.click(screen.getByTestId("node-panel-run-dialog-submit"));
    });
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("node-panel-run-dialog-submit")).toBeTruthy();
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
