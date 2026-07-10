import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, afterEach } from "vitest";

import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
  component: "start",
  configuration: {
    templates: [{ name: "default", payload: { issue: { number: 0 } } }],
  },
};

const RENDER: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "service", label: "Service" },
    { field: "status", format: "status" },
  ],
  rowActions: [
    {
      kind: "trigger",
      label: "Redeploy",
      node: "start",
      show: 'status == "failed"',
    },
  ],
};

const TWO_FAILED_ROWS = [
  { id: "mem-1", namespace: "environments", service: "api", status: "failed", pr_number: "42" },
  { id: "mem-2", namespace: "environments", service: "web", status: "failed", pr_number: "7" },
];

function seedRunsCache(queryClient: QueryClient, runs: CanvasesCanvasRun[]) {
  queryClient.setQueryData(canvasKeys.infiniteRuns("canvas-1", { states: ["STATE_STARTED"] }), {
    pages: [
      {
        runs,
        totalCount: runs.length,
        lastTimestamp: undefined,
      },
    ],
    pageParams: [undefined],
  });
}

function renderTableWithCache({
  queryClient,
  rows,
  onTriggerNode,
}: {
  queryClient: QueryClient;
  rows: unknown[];
  onTriggerNode?: (nodeId: string, opts?: { hookName?: string; successLabel?: string }) => Promise<void>;
}) {
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[START_NODE]}
          canRunNodes
          onTriggerNode={onTriggerNode ? (id, opts) => void onTriggerNode(id, opts) : undefined}
        >
          <WidgetTable render={RENDER} rows={rows} isLoading={false} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable row actions - in-flight gating", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps row-action buttons enabled when a STATE_STARTED run is in cache but was not initiated from a tracked row", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    seedRunsCache(queryClient, [
      {
        id: "run-1",
        state: "STATE_STARTED",
        rootEvent: { nodeId: START_NODE.id, root: true },
      },
    ]);

    renderTableWithCache({ queryClient, rows: TWO_FAILED_ROWS });

    const buttons = screen.getAllByTestId("widget-row-action-start");
    expect(buttons).toHaveLength(2);

    await waitFor(() => {
      for (const button of buttons) expect(button).not.toBeDisabled();
    });
  });

  it("does not disable buttons for runs that target a different trigger node", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    seedRunsCache(queryClient, [
      {
        id: "run-1",
        state: "STATE_STARTED",
        rootEvent: { nodeId: "some-other-node", root: true },
      },
    ]);

    renderTableWithCache({ queryClient, rows: TWO_FAILED_ROWS });

    const buttons = screen.getAllByTestId("widget-row-action-start");
    await waitFor(() => {
      for (const button of buttons) expect(button).not.toBeDisabled();
    });
  });

  it("locks only the clicked row during the submission window; siblings stay enabled", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    let resolveTrigger: () => void = () => {};
    const onTrigger = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveTrigger = resolve;
        }),
    );

    renderTableWithCache({ queryClient, rows: TWO_FAILED_ROWS, onTriggerNode: onTrigger });

    const buttons = screen.getAllByTestId("widget-row-action-start");
    expect(buttons).toHaveLength(2);

    await act(async () => {
      fireEvent.click(buttons[0]);
    });

    await waitFor(() => {
      expect(buttons[0]).toBeDisabled();
      expect(buttons[0].getAttribute("data-disabled-reason")).toBe("submitting");
    });
    expect(buttons[1]).not.toBeDisabled();

    await act(async () => {
      resolveTrigger();
      await Promise.resolve();
    });

    expect(buttons[0]).toBeDisabled();
    expect(buttons[1]).not.toBeDisabled();

    await act(async () => {
      vi.advanceTimersByTime(1600);
    });

    await waitFor(() => {
      expect(buttons[0]).not.toBeDisabled();
    });
    expect(buttons[1]).not.toBeDisabled();
  });

  it("keeps the originating row locked while its STATE_STARTED run is in cache, and unlocks when the run finishes", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const onTrigger = vi.fn().mockResolvedValue(undefined);

    renderTableWithCache({ queryClient, rows: TWO_FAILED_ROWS, onTriggerNode: onTrigger });

    const buttons = screen.getAllByTestId("widget-row-action-start");

    await act(async () => {
      fireEvent.click(buttons[0]);
      await Promise.resolve();
    });

    act(() => {
      seedRunsCache(queryClient, [
        {
          id: "run-1",
          state: "STATE_STARTED",
          rootEvent: { nodeId: START_NODE.id, root: true },
        },
      ]);
    });

    await waitFor(() => {
      expect(buttons[0]).toBeDisabled();
      expect(buttons[0].getAttribute("data-disabled-reason")).toBe("run-in-flight");
    });
    expect(buttons[1]).not.toBeDisabled();

    act(() => {
      seedRunsCache(queryClient, []);
    });

    await waitFor(() => {
      for (const button of buttons) expect(button).not.toBeDisabled();
    });
  });
});
