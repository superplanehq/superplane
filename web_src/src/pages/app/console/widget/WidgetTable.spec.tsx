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
        <ConsoleContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[START_NODE]}
          canRunNodes={canRunNodes}
          onTriggerNode={onTriggerNode ? (id, opts) => void onTriggerNode(id, opts) : undefined}
        >
          <WidgetTable render={RENDER} rows={ROWS} isLoading={false} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable row styles — background tone", () => {
  const ROW_STYLE_ROWS = [
    { id: "row-error", service: "api", status: "error" },
    { id: "row-deploying", service: "web", status: "deploying" },
    { id: "row-ok", service: "worker", status: "passed" },
  ];

  function renderWithRowStyles(rowStyles: WidgetTableRender["rowStyles"]) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable
              render={{
                kind: "table",
                columns: [
                  { field: "service", label: "Service" },
                  { field: "status", label: "Status" },
                ],
                rowStyles,
              }}
              rows={ROW_STYLE_ROWS}
              isLoading={false}
            />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  it("applies the first matching tone class to the row and skips non-matches", () => {
    const view = renderWithRowStyles([
      { field: "status", op: "eq", value: "error", tone: "red-soft" },
      { field: "status", op: "eq", value: "deploying", tone: "orange-soft" },
    ]);
    const rows = view.container.querySelectorAll("table tbody tr");
    expect(rows).toHaveLength(3);
    expect(rows[0].className).toContain("bg-red-50");
    expect(rows[0].getAttribute("data-row-tone")).toBe("true");
    expect(rows[1].className).toContain("bg-orange-50");
    // Row 3 doesn't match any rule, so no tone marker and no tone class.
    expect(rows[2].getAttribute("data-row-tone")).toBeNull();
    expect(rows[2].className).not.toMatch(/(?:^|\s)bg-(red|orange|yellow|sky|emerald|slate)-(50|100)(?:\s|$)/);
    // Default hover wash should remain on untinted rows so they keep the
    // existing hover affordance.
    expect(rows[2].className).toContain("hover:bg-slate-50/60");
  });

  it("first matching rule wins when multiple rules match the same row", () => {
    const view = renderWithRowStyles([
      { field: "status", op: "contains", value: "err", tone: "red" },
      { field: "status", op: "eq", value: "error", tone: "green" },
    ]);
    const firstRow = view.container.querySelector("table tbody tr");
    expect(firstRow).not.toBeNull();
    expect(firstRow!.className).toContain("bg-red-100");
    expect(firstRow!.className).not.toContain("bg-green-");
  });
});

describe("WidgetTable link column href", () => {
  function renderLink(tableRender: WidgetTableRender, rows: unknown[]) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable render={tableRender} rows={rows} isLoading={false} />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  it("resolves {{ field }} expressions in href, rendering the cell value as the link text", () => {
    const view = renderLink(
      {
        kind: "table",
        columns: [{ field: "prNumber", format: "link", href: "{{ prUrl }}" }],
      },
      [{ id: "row-1", prNumber: 42, prUrl: "https://github.com/acme/core/pull/42" }],
    );
    const anchor = view.container.querySelector("table tbody tr a");
    expect(anchor).not.toBeNull();
    expect(anchor!.getAttribute("href")).toBe("https://github.com/acme/core/pull/42");
    expect(anchor!.textContent).toBe("42");
    view.unmount();
  });

  it("resolves mixed {{ }} templates with literal text in href", () => {
    const view = renderLink(
      {
        kind: "table",
        columns: [
          { field: "prNumber", format: "link", href: "https://github.com/{{ org }}/{{ repo }}/pull/{{ prNumber }}" },
        ],
      },
      [{ id: "row-1", prNumber: 7, org: "acme", repo: "core" }],
    );
    const anchor = view.container.querySelector("table tbody tr a");
    expect(anchor!.getAttribute("href")).toBe("https://github.com/acme/core/pull/7");
    expect(anchor!.textContent).toBe("7");
    view.unmount();
  });

  it("keeps legacy single-brace {field} placeholders working", () => {
    const view = renderLink(
      {
        kind: "table",
        columns: [{ field: "service", format: "link", href: "/services/{service}" }],
      },
      [{ id: "row-1", service: "api" }],
    );
    const anchor = view.container.querySelector("table tbody tr a");
    expect(anchor!.getAttribute("href")).toBe("/services/api");
    view.unmount();
  });
});

describe("WidgetTable column formatting", () => {
  it("renders status columns as colored pills and badge columns as neutral tags", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const renderWithFormat = (format: "status" | "badge") =>
      render(
        <MemoryRouter>
          <QueryClientProvider client={queryClient}>
            <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
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
            </ConsoleContextProvider>
          </QueryClientProvider>
        </MemoryRouter>,
      );

    const statusView = renderWithFormat("status");
    const statusPill = statusView.container.querySelector("table tbody tr td:nth-child(2) span");
    expect(statusPill).not.toBeNull();
    expect(statusPill!.textContent).toBe("failed");
    expect(statusPill!.className).toContain("bg-red-500");
    expect(statusPill!.className).toContain("text-white");
    statusView.unmount();

    const badgeView = renderWithFormat("badge");
    const badgePill = badgeView.container.querySelector("table tbody tr td:nth-child(2) span");
    expect(badgePill).not.toBeNull();
    expect(badgePill!.textContent).toBe("failed");
    expect(badgePill!.className).toContain("bg-transparent");
    expect(badgePill!.className).toContain("outline-slate-950/15");
    expect(badgePill!.className).toContain("text-slate-700");
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

describe("WidgetTable row actions — manual-run gating", () => {
  const PR_NODE: SuperplaneComponentsNode = {
    id: "pr-id",
    name: "on-pr",
    type: "TYPE_TRIGGER",
    component: "github.pullRequest",
  };
  const START_NODE_WITH_COMPONENT: SuperplaneComponentsNode = { ...START_NODE, component: "start" };
  const EVENT_ROW_RENDER: WidgetTableRender = {
    kind: "table",
    columns: [{ field: "service", label: "Service" }],
    rowActions: [
      { kind: "trigger", label: "Redeploy", node: "start" },
      { kind: "trigger", label: "Reopen PR", node: "on-pr" },
    ],
  };

  function renderWithManualRunCatalog(manualRunTriggers: ReadonlySet<string>) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider
            canvasId="canvas-1"
            organizationId="org-1"
            nodes={[START_NODE_WITH_COMPONENT, PR_NODE]}
            canRunNodes
            manualRunTriggers={manualRunTriggers}
            onTriggerNode={() => undefined}
          >
            <WidgetTable render={EVENT_ROW_RENDER} rows={ROWS} isLoading={false} />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  it("hides row actions whose node is not manual-runnable", () => {
    renderWithManualRunCatalog(new Set(["start"]));
    expect(screen.queryAllByTestId("widget-row-action-start")).toHaveLength(2);
    expect(screen.queryByTestId("widget-row-action-on-pr")).toBeNull();
  });

  it("keeps manual-run actions visible when the catalog lists them", () => {
    renderWithManualRunCatalog(new Set(["start", "github.pullRequest"]));
    expect(screen.queryAllByTestId("widget-row-action-start")).toHaveLength(2);
    expect(screen.queryAllByTestId("widget-row-action-on-pr")).toHaveLength(2);
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
            <ConsoleContextProvider
              canvasId="canvas-1"
              organizationId="org-1"
              nodes={[START_NODE]}
              canRunNodes
              onTriggerNode={(id, opts) => void onTrigger(id, opts)}
            >
              <WidgetTable render={canvasRender} rows={ROWS} isLoading={false} />
            </ConsoleContextProvider>
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

    // Row payload templates are merged into run-hook parameters so authors
    // can wire per-row values (e.g. pr_number) into the trigger payload.
    const params = screen.getByTestId("widget-row-action-start-parameters");
    expect(params.textContent).toContain('"template": "default"');
    expect(params.textContent).toContain('"number": "42"');
    expect(params.getAttribute("class")).toContain("overflow-x-auto");
    expect(params.getAttribute("class")).toContain("whitespace-pre");
    expect(params.getAttribute("class")).not.toContain("break-all");
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
