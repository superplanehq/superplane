import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, describe, it, expect, vi } from "vitest";

import type { ConsolePanel } from "@/hooks/useCanvasData";
import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { NodesPanelCard } from "./NodesPanelCard";

// Same runs-query stub as `NodesPanelCard.spec.tsx`: mutate `mockInFlight`
// to simulate a `STATE_STARTED` canvas run for the panel's trigger node.
let mockInFlight = new Set<string>();
vi.mock("./widget/useInFlightTriggers", () => ({
  useInFlightTriggers: () => ({ inFlight: mockInFlight, isLoading: false }),
}));

const NODE_WITH_PARAMS: SuperplaneComponentsNode = {
  id: "node-1",
  name: "deploy-prod",
  type: "TYPE_TRIGGER",
  component: "start",
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

const NODE_NO_PARAMS: SuperplaneComponentsNode = {
  id: "node-1",
  name: "deploy-prod",
  type: "TYPE_TRIGGER",
  component: "start",
  configuration: {
    templates: [{ name: "manual", payload: { reason: "console" } }],
  },
};

/** Panel opting into concurrent run submissions at the widget level. */
function concurrentPanel(entries: Array<Record<string, unknown>>): ConsolePanel {
  return {
    id: "key-nodes",
    type: "nodes",
    content: { title: "Key nodes", allowConcurrentRuns: true, nodes: entries },
  };
}

function renderPanel(nodes: SuperplaneComponentsNode[], panel: ConsolePanel, onTriggerNode = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <ConsoleContextProvider
        canvasId="canvas-1"
        organizationId="org-1"
        nodes={nodes}
        canRunNodes
        onTriggerNode={onTriggerNode}
      >
        <NodesPanelCard panel={panel} readOnly onDelete={() => undefined} onChange={() => undefined} />
      </ConsoleContextProvider>
    </QueryClientProvider>,
  );
  return onTriggerNode;
}

afterEach(() => {
  mockInFlight = new Set();
});

describe("NodesPanelCard — concurrent runs option", () => {
  it("keeps the Run button enabled while the trigger has a STATE_STARTED run", () => {
    mockInFlight = new Set(["node-1"]);
    renderPanel([NODE_NO_PARAMS], concurrentPanel([{ node: "deploy-prod", showRun: true }]));
    const runButton = screen.getByTestId("node-panel-run");
    expect(runButton).not.toBeDisabled();
    expect(runButton).not.toHaveAttribute("data-disabled-reason");
  });

  it("fires an additional run while a run is already in flight", async () => {
    mockInFlight = new Set(["node-1"]);
    const onTrigger = renderPanel([NODE_NO_PARAMS], concurrentPanel([{ node: "deploy-prod", showRun: true }]));
    await act(async () => {
      fireEvent.click(screen.getByTestId("node-panel-run"));
    });
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(1));
  });

  it("does not lock sibling entries targeting the same trigger while submitting", async () => {
    let resolveTrigger: (() => void) | undefined;
    const onTrigger = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveTrigger = resolve;
        }),
    );
    renderPanel(
      [NODE_NO_PARAMS],
      concurrentPanel([
        { node: "deploy-prod", showRun: true },
        { node: "deploy-prod", showRun: true },
      ]),
      onTrigger,
    );
    const [first, second] = screen.getAllByTestId("nodes-panel-row-run");
    fireEvent.click(first);
    // The sibling stays clickable even though the first submission is still
    // in flight — concurrent submissions are explicitly allowed.
    expect(second).not.toBeDisabled();
    fireEvent.click(second);
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(2));
    await act(async () => {
      resolveTrigger?.();
    });
  });

  it("keeps the inline submit enabled during an active run", () => {
    mockInFlight = new Set(["node-1"]);
    renderPanel(
      [NODE_WITH_PARAMS],
      concurrentPanel([{ node: "deploy-prod", showRun: true, triggerName: "manual", formMode: "inline" }]),
    );
    expect(screen.getByTestId("node-panel-run-inline-submit")).not.toBeDisabled();
  });
});
