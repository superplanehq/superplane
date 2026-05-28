import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import type { DashboardPanel } from "@/hooks/useCanvasData";
import type { SuperplaneComponentsNode } from "@/api-client";

import { DashboardContextProvider } from "./DashboardContextProvider";
import type { DashboardTriggerOptions } from "./DashboardContext";
import { NodePanelCard } from "./NodePanelCard";

const NODE_NO_PARAMS: SuperplaneComponentsNode = {
  id: "node-1",
  name: "deploy-prod",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "manual", payload: { reason: "console" } }],
  },
};

const PANEL: DashboardPanel = {
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
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: DashboardTriggerOptions) => void;
}) {
  return render(
    <DashboardContextProvider
      canvasId="canvas-1"
      organizationId="org-1"
      nodes={[NODE_NO_PARAMS]}
      canRunNodes={canRunNodes}
      onTriggerNode={onTriggerNode}
    >
      <NodePanelCard panel={PANEL} readOnly onDelete={() => undefined} onChange={() => undefined} />
    </DashboardContextProvider>,
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

  it("disables the Run button when the viewer cannot run nodes", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: false, onTriggerNode: onTrigger });
    const runButton = screen.getByTestId("node-panel-run");
    expect(runButton).toBeDisabled();
    fireEvent.click(runButton);
    expect(onTrigger).not.toHaveBeenCalled();
  });
});
