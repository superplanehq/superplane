import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, describe, it, expect, vi } from "vitest";

import type { ConsolePanel } from "@/hooks/useCanvasData";
import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import type { ConsoleTriggerOptions } from "./ConsoleContext";
import { NodesPanelCard } from "./NodesPanelCard";

// The run button lock now subscribes to the runs query, so tests need a
// QueryClient in scope even if the query never actually fires (fake
// canvas id + no `useCanvasWebsocket`). Tests can mutate `mockInFlight`
// to simulate a `STATE_STARTED` canvas run for the panel's trigger.
let mockInFlight = new Set<string>();
vi.mock("./widget/useInFlightTriggers", () => ({
  useInFlightTriggers: () => ({ inFlight: mockInFlight, isLoading: false }),
}));

const NODE: SuperplaneComponentsNode = {
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

const NODE_ACTION: SuperplaneComponentsNode = {
  id: "node-2",
  name: "publish-artifact",
  type: "TYPE_ACTION",
};

const PR_TRIGGER: SuperplaneComponentsNode = {
  id: "pr-id",
  name: "on-pr",
  type: "TYPE_TRIGGER",
  component: "github.pullRequest",
};

/**
 * Single-entry panel — renders using the compact centered layout with
 * `node-panel-run` test ids (equivalent to the pre-merge single-node card).
 */
function singleNodePanel(overrides: Partial<Record<string, unknown>> = {}): ConsolePanel {
  return {
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
          ...overrides,
        },
      ],
    },
  };
}

/** Multi-entry panel — renders as a row list. */
function multiNodePanel(entries: Array<Record<string, unknown>>): ConsolePanel {
  return {
    id: "key-nodes",
    type: "nodes",
    content: { title: "Key nodes", nodes: entries },
  };
}

function renderPanel({
  canRunNodes,
  onTriggerNode,
  nodes = [NODE],
  panel = singleNodePanel(),
  manualRunTriggers,
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: ConsoleTriggerOptions) => void;
  nodes?: SuperplaneComponentsNode[];
  panel?: ConsolePanel;
  manualRunTriggers?: ReadonlySet<string>;
}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  // Fresh element per call — reusing the same element reference would let
  // React bail out of re-rendering, hiding `mockInFlight` changes.
  const buildUi = () => (
    <QueryClientProvider client={queryClient}>
      <ConsoleContextProvider
        canvasId="canvas-1"
        organizationId="org-1"
        nodes={nodes}
        canRunNodes={canRunNodes}
        manualRunTriggers={manualRunTriggers}
        onTriggerNode={onTriggerNode}
      >
        <NodesPanelCard panel={panel} readOnly onDelete={() => undefined} onChange={() => undefined} />
      </ConsoleContextProvider>
    </QueryClientProvider>
  );
  const view = render(buildUi());
  // Re-render the same tree — lets tests mutate `mockInFlight` mid-test to
  // simulate a run starting elsewhere while a dialog is already open.
  const rerenderPanel = () => view.rerender(buildUi());
  return { ...view, rerenderPanel };
}

afterEach(() => {
  mockInFlight = new Set();
});

describe("NodesPanelCard — single-entry layout", () => {
  it("opens the confirm dialog for a template with input fields", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    expect(onTrigger).not.toHaveBeenCalled();
    expect(screen.getByTestId("node-panel-run-dialog-submit")).toBeTruthy();
  });

  it("submits the merged parameters via ctx.onTriggerNode on confirm", async () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    const branch = screen.getByLabelText("branch") as HTMLInputElement;
    fireEvent.change(branch, { target: { value: "release/v3" } });
    await act(async () => {
      fireEvent.click(screen.getByTestId("node-panel-run-dialog-submit"));
    });
    await waitFor(() => expect(onTrigger).toHaveBeenCalledTimes(1));
    expect(onTrigger).toHaveBeenCalledWith("node-1", {
      hookName: "run",
      templateName: "manual",
      parameters: { template: "manual", branch: "release/v3" },
    });
  });

  it("triggers immediately for a parameter-less template when confirmation is not required", async () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, nodes: [NODE_NO_PARAMS] });
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

  it("disables the Run button while the direct trigger is in flight, then re-enables after the grace window", async () => {
    let resolveTrigger: (() => void) | undefined;
    const onTrigger = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveTrigger = resolve;
        }),
    );
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, nodes: [NODE_NO_PARAMS] });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    await waitFor(() => expect(screen.getByTestId("node-panel-run")).toBeDisabled());

    // Real timers up to this point so `waitFor` polls work. Switch to fake
    // timers to fast-forward past the 1500ms submission grace window.
    vi.useFakeTimers();
    try {
      await act(async () => {
        resolveTrigger?.();
      });
      // The button stays disabled during the grace window, mirroring the
      // table row-action lock behavior.
      expect(screen.getByTestId("node-panel-run")).toBeDisabled();
      await act(async () => {
        vi.advanceTimersByTime(1500);
      });
      expect(screen.getByTestId("node-panel-run")).not.toBeDisabled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("disables the Run button when the viewer cannot run nodes", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: false, onTriggerNode: onTrigger });
    const runButton = screen.getByTestId("node-panel-run");
    expect(runButton).toBeDisabled();
    fireEvent.click(runButton);
    expect(onTrigger).not.toHaveBeenCalled();
  });

  it("does not render the Run button for non-trigger nodes even when showRun is set", () => {
    renderPanel({
      canRunNodes: true,
      nodes: [NODE_ACTION],
      panel: multiNodePanel([{ node: "publish-artifact", showRun: true }]),
    });
    // Single-entry -> compact layout; no Run should render for an action node.
    expect(screen.queryByTestId("node-panel-run")).toBeNull();
  });

  it("hides the Run button for a trigger node whose component is not manual-runnable", () => {
    renderPanel({
      canRunNodes: true,
      nodes: [PR_TRIGGER],
      manualRunTriggers: new Set(["start"]),
      panel: multiNodePanel([{ node: "on-pr", showRun: true }]),
    });
    expect(screen.queryByTestId("node-panel-run")).toBeNull();
  });

  it("renders the label override instead of the resolved node name", () => {
    renderPanel({
      canRunNodes: true,
      panel: singleNodePanel({ label: "Ship to prod" }),
    });
    expect(screen.getByTestId("node-panel-name").textContent).toBe("Ship to prod");
  });

  it("renders a legacy type: 'node' panel via the same compact layout", () => {
    const legacyPanel: ConsolePanel = {
      id: "legacy-deploy",
      type: "node",
      content: {
        title: "Deploy",
        node: "deploy-prod",
        showRun: true,
        triggerName: "manual",
      },
    };
    renderPanel({ canRunNodes: true, nodes: [NODE_NO_PARAMS], panel: legacyPanel });
    expect(screen.getByTestId("node-panel-name").textContent).toBe("deploy-prod");
    expect(screen.getByTestId("node-panel-run")).toBeInTheDocument();
    // Row-layout locators must not appear for a one-entry panel.
    expect(screen.queryByTestId("nodes-panel-row")).toBeNull();
  });

  it("renders the compact unconfigured state for a legacy type: 'node' panel without a node", () => {
    const legacyPanel: ConsolePanel = {
      id: "legacy-empty",
      type: "node",
      content: { title: "Deploy" },
    };
    renderPanel({ canRunNodes: true, panel: legacyPanel });
    expect(screen.getByText(/pick a node from the editor/i)).toBeInTheDocument();
    expect(screen.queryByText(/add nodes from the editor/i)).toBeNull();
  });
});

describe("NodesPanelCard — in-flight lock", () => {
  it("disables the single-entry Run button while its trigger has a STATE_STARTED run", () => {
    mockInFlight = new Set(["node-1"]);
    renderPanel({ canRunNodes: true, onTriggerNode: vi.fn(), nodes: [NODE_NO_PARAMS] });
    const runButton = screen.getByTestId("node-panel-run");
    expect(runButton).toBeDisabled();
    expect(runButton).toHaveAttribute("data-disabled-reason", "run-in-flight");
    expect(runButton.getAttribute("title")).toMatch(/already in progress/i);
  });

  it("disables the multi-entry Run button while its trigger is running elsewhere", () => {
    mockInFlight = new Set(["node-1"]);
    renderPanel({
      canRunNodes: true,
      onTriggerNode: vi.fn(),
      nodes: [NODE_NO_PARAMS],
      panel: multiNodePanel([
        { node: "deploy-prod", showRun: true },
        { node: "deploy-prod", showRun: true },
      ]),
    });
    const buttons = screen.getAllByTestId("nodes-panel-row-run");
    // Two identical-target entries — both should reflect the same in-flight state.
    for (const button of buttons) {
      expect(button).toBeDisabled();
      expect(button).toHaveAttribute("data-disabled-reason", "run-in-flight");
    }
  });

  it("blocks a confirm dialog opened before a run started from firing a duplicate", () => {
    const onTrigger = vi.fn();
    const { rerenderPanel } = renderPanel({ canRunNodes: true, onTriggerNode: onTrigger });
    fireEvent.click(screen.getByTestId("node-panel-run"));
    const submit = screen.getByTestId("node-panel-run-dialog-submit");
    expect(submit).not.toBeDisabled();

    // A run for the same trigger starts elsewhere while the dialog is open.
    mockInFlight = new Set(["node-1"]);
    rerenderPanel();

    expect(submit).toBeDisabled();
    fireEvent.click(submit);
    expect(onTrigger).not.toHaveBeenCalled();
  });
});

describe("NodesPanelCard — multi-entry layout", () => {
  const MULTI_PANEL = multiNodePanel([
    { node: "deploy-prod", description: "Deploys production", showRun: true, triggerName: "manual" },
    { node: "publish-artifact" },
  ]);

  it("renders the row list and its Run button on the trigger row", () => {
    const onTrigger = vi.fn();
    renderPanel({ canRunNodes: true, onTriggerNode: onTrigger, nodes: [NODE, NODE_ACTION], panel: MULTI_PANEL });
    expect(screen.getAllByTestId("nodes-panel-row")).toHaveLength(2);
    expect(screen.getByTestId("nodes-panel-row-run")).toBeInTheDocument();
  });

  it("hides row Run buttons for non-manual-runnable trigger nodes", () => {
    renderPanel({
      canRunNodes: true,
      nodes: [NODE, PR_TRIGGER],
      manualRunTriggers: new Set(["start"]),
      panel: multiNodePanel([
        { node: "deploy-prod", showRun: true },
        { node: "on-pr", showRun: true },
      ]),
    });
    expect(screen.getAllByTestId("nodes-panel-row-run")).toHaveLength(1);
  });
});
