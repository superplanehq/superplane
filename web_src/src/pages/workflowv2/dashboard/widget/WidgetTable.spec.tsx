import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, afterEach } from "vitest";

import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { DashboardContextProvider } from "../DashboardContextProvider";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "default", payload: { issue: { number: 0 } } }],
  },
};

const ROWS = [
  { id: "mem-1", namespace: "environments", service: "api", status: "failed", pr_number: "42" },
  { id: "mem-2", namespace: "environments", service: "web", status: "passed", pr_number: "7" },
];

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

function renderTable({
  canRunNodes,
  onTriggerNode,
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: { hookName?: string; successLabel?: string }) => Promise<void>;
}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <DashboardContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[START_NODE]}
          canRunNodes={canRunNodes}
          onTriggerNode={onTriggerNode ? (id, opts) => void onTriggerNode(id, opts) : undefined}
        >
          <WidgetTable render={RENDER} rows={ROWS} isLoading={false} />
        </DashboardContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable column formatting", () => {
  it("renders status and badge columns as pills with the same classes", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const renderWithFormat = (format: "status" | "badge") =>
      render(
        <MemoryRouter>
          <QueryClientProvider client={queryClient}>
            <DashboardContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
              <WidgetTable
                render={{
                  kind: "table",
                  columns: [
                    { field: "service", label: "Service" },
                    { field: "status", format },
                  ],
                }}
                rows={ROWS}
                isLoading={false}
              />
            </DashboardContextProvider>
          </QueryClientProvider>
        </MemoryRouter>,
      );

    const statusView = renderWithFormat("status");
    const statusPill = statusView.container.querySelector("table tbody tr td:nth-child(2) span");
    expect(statusPill).not.toBeNull();
    expect(statusPill!.textContent).toBe("failed");
    const statusClass = statusPill!.getAttribute("class") ?? "";
    statusView.unmount();

    const badgeView = renderWithFormat("badge");
    const badgePill = badgeView.container.querySelector("table tbody tr td:nth-child(2) span");
    expect(badgePill).not.toBeNull();
    expect(badgePill!.textContent).toBe("failed");
    expect(badgePill!.getAttribute("class")).toBe(statusClass);
  });
});

describe("WidgetTable row actions — permission gating", () => {
  it("invokes the trigger callback when canRunNodes is true", async () => {
    const onTrigger = vi.fn().mockResolvedValue(undefined);
    renderTable({ canRunNodes: true, onTriggerNode: onTrigger });
    const triggers = screen.getAllByTestId("widget-row-action-start");
    expect(triggers).toHaveLength(1);
    expect(triggers[0]).not.toBeDisabled();
    await act(async () => {
      fireEvent.click(triggers[0]);
    });
    expect(onTrigger).toHaveBeenCalledWith(
      "start-id",
      expect.objectContaining({
        hookName: "run",
        successLabel: "Redeploy",
      }),
    );
  });

  it("renders trigger disabled when canRunNodes is false", () => {
    const onTrigger = vi.fn();
    renderTable({ canRunNodes: false, onTriggerNode: onTrigger });
    const trigger = screen.getByTestId("widget-row-action-start");
    expect(trigger).toBeDisabled();
    fireEvent.click(trigger);
    expect(onTrigger).not.toHaveBeenCalled();
  });

  it("evaluates per-row show expressions", () => {
    renderTable({ canRunNodes: true });
    expect(screen.queryAllByTestId("widget-row-action-start")).toHaveLength(1);
  });
});

describe("WidgetTable row actions — confirm dialog preview", () => {
  it("shows resolved trigger node, template, and run hook parameters", () => {
    const onTrigger = vi.fn().mockResolvedValue(undefined);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const renderWithConfirm = (canvasRender: WidgetTableRender) =>
      render(
        <MemoryRouter>
          <QueryClientProvider client={queryClient}>
            <DashboardContextProvider
              canvasId="canvas-1"
              organizationId="org-1"
              nodes={[START_NODE]}
              canRunNodes
              onTriggerNode={(id, opts) => void onTrigger(id, opts)}
            >
              <WidgetTable render={canvasRender} rows={ROWS} isLoading={false} />
            </DashboardContextProvider>
          </QueryClientProvider>
        </MemoryRouter>,
      );

    const withConfirm: WidgetTableRender = {
      ...RENDER,
      rowActions: [
        {
          kind: "trigger",
          label: "Redeploy",
          node: "start",
          show: 'status == "failed"',
          confirm: "Redeploy {{ service }}?",
          payload: { "issue.number": "{{ pr_number }}" },
        },
      ],
    };
    renderWithConfirm(withConfirm);

    fireEvent.click(screen.getByTestId("widget-row-action-start"));

    // Confirm message interpolated against row.
    expect(screen.getByText("Redeploy api?")).toBeTruthy();

    // Trigger label resolved (label/name plus id).
    const preview = screen.getByTestId("widget-row-action-start-preview");
    expect(preview.textContent).toMatch(/start/);
    expect(preview.textContent).toMatch(/start-id/);

    // Hook + template names rendered.
    expect(preview.textContent).toMatch(/run/);
    expect(preview.textContent).toMatch(/default/);

    // Manual run hooks pass template selection only; row payload templates are not merged.
    const params = screen.getByTestId("widget-row-action-start-parameters");
    expect(params.textContent).toContain('"template": "default"');
    expect(params.textContent).not.toContain('"number"');
  });
});

/* -------------------------------------------------------------------------- */
/*  In-flight gating                                                           */
/* -------------------------------------------------------------------------- */

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
        <DashboardContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[START_NODE]}
          canRunNodes
          onTriggerNode={onTriggerNode ? (id, opts) => void onTriggerNode(id, opts) : undefined}
        >
          <WidgetTable render={RENDER} rows={rows} isLoading={false} />
        </DashboardContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable row actions — in-flight gating", () => {
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
    // Without a recorded source row, the per-row lock model treats the
    // in-flight run as "we don't know which row started it", so siblings
    // stay clickable. This is the explicit row-only locking contract.
    await waitFor(() => {
      for (const b of buttons) expect(b).not.toBeDisabled();
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
      for (const b of buttons) expect(b).not.toBeDisabled();
    });
  });

  it("locks only the clicked row during the submission window; siblings stay enabled", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    // Deferred trigger so we can observe the in-flight window while the
    // promise is unresolved, then close it and watch the grace timer run.
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

    // Click the first row's action; only that row should disable. The
    // sibling row stays clickable because the per-row lock model scopes
    // submission state to the row that initiated it.
    await act(async () => {
      fireEvent.click(buttons[0]);
    });

    await waitFor(() => {
      expect(buttons[0]).toBeDisabled();
      expect(buttons[0].getAttribute("data-disabled-reason")).toBe("submitting");
    });
    expect(buttons[1]).not.toBeDisabled();

    // Resolve the deferred trigger — the grace timer keeps the clicked row
    // disabled for ~1500 ms while we wait for the websocket-driven runs
    // cache to catch up.
    await act(async () => {
      resolveTrigger();
      await Promise.resolve();
    });

    expect(buttons[0]).toBeDisabled();
    expect(buttons[1]).not.toBeDisabled();

    // After the grace window the lock should release.
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

    // Click the first row to record the rowKey → trigger mapping.
    await act(async () => {
      fireEvent.click(buttons[0]);
      await Promise.resolve();
    });

    // Now seed the runs cache so the trigger appears in flight. The first
    // row stays locked via the in-flight mapping; the sibling remains
    // clickable because the mapping points to row 0.
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

    // Simulate the websocket invalidation flow: the runs query now returns
    // an empty page (the run finished and is no longer STATE_STARTED).
    act(() => {
      seedRunsCache(queryClient, []);
    });

    await waitFor(() => {
      for (const b of buttons) expect(b).not.toBeDisabled();
    });
  });
});
