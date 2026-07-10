import { render, screen, fireEvent, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";
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
    expect(statusPill!.className).toContain("uppercase");
    expect(statusPill!.className).toContain("tracking-wide");
    statusView.unmount();

    const badgeView = renderWithFormat("badge");
    const badgePill = badgeView.container.querySelector("table tbody tr td:nth-child(2) span");
    expect(badgePill).not.toBeNull();
    expect(badgePill!.textContent).toBe("failed");
    expect(badgePill!.className).toContain("bg-red-500");
    expect(badgePill!.className).toContain("text-[10px]");
    expect(badgePill!.className).toContain("font-semibold");
  });

  it("renders avatar columns with the deployer name in a tooltip", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable
              render={{
                kind: "table",
                columns: [
                  {
                    field: "author",
                    format: "avatar",
                    avatarCommitterField: "committer",
                    label: "Who",
                  },
                ],
              }}
              rows={[
                {
                  author: { name: "Pedro Leão", username: "forestileao" },
                  committer: { name: "Pedro Leão" },
                },
              ]}
              isLoading={false}
            />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const avatar = view.container.querySelector('[data-slot="avatar"] img');
    expect(avatar).not.toBeNull();
    expect(avatar!.getAttribute("src")).toBe("https://github.com/forestileao.png");
    expect(screen.queryByText("Pedro Leão")).not.toBeInTheDocument();
    view.unmount();
  });

  it("renders avatar columns for plain username strings", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable
              render={{
                kind: "table",
                columns: [{ field: "author", label: "Author", format: "avatar" }],
              }}
              rows={[{ author: "forestileao" }]}
              isLoading={false}
            />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const avatar = view.container.querySelector('[data-slot="avatar"] img');
    expect(avatar).not.toBeNull();
    expect(avatar!.getAttribute("src")).toBe("https://github.com/forestileao.png");
    view.unmount();
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
    component: "github.onPullRequest",
  };
  const SCHEDULE_NODE: SuperplaneComponentsNode = {
    id: "schedule-id",
    name: "nightly",
    type: "TYPE_TRIGGER",
    component: "schedule",
  };
  const EVENT_ROW_RENDER: WidgetTableRender = {
    kind: "table",
    columns: [{ field: "service", label: "Service" }],
    rowActions: [
      { kind: "trigger", label: "Redeploy", node: "start" },
      { kind: "trigger", label: "Reopen PR", node: "on-pr" },
      { kind: "trigger", label: "Run now", node: "nightly" },
    ],
  };

  function renderWithNodes(nodes: SuperplaneComponentsNode[]) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider
            canvasId="canvas-1"
            organizationId="org-1"
            nodes={nodes}
            canRunNodes
            onTriggerNode={() => undefined}
          >
            <WidgetTable render={EVENT_ROW_RENDER} rows={ROWS} isLoading={false} />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  it("hides row actions whose trigger is event-driven (not start/schedule)", () => {
    renderWithNodes([START_NODE, PR_NODE, SCHEDULE_NODE]);
    expect(screen.queryAllByTestId("widget-row-action-start")).toHaveLength(2);
    expect(screen.queryAllByTestId("widget-row-action-nightly")).toHaveLength(2);
    expect(screen.queryByTestId("widget-row-action-on-pr")).toBeNull();
  });
});

describe("WidgetTable trend columns", () => {
  function renderTrend({
    render: tableRender,
    rows,
    hasMore,
  }: {
    render: WidgetTableRender;
    rows: unknown[];
    hasMore?: boolean;
  }) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable render={tableRender} rows={rows} isLoading={false} hasMore={hasMore} />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  const TREND_RENDER: WidgetTableRender = {
    kind: "table",
    columns: [
      { field: "id", label: "Deploy" },
      {
        field: "durationMs",
        label: "Trend",
        format: "trend",
        trendBetter: "down",
        trendDisplay: "percent",
      },
    ],
  };

  const TREND_ROWS = [
    { id: "d-3", durationMs: 900 },
    { id: "d-2", durationMs: 1000 },
    { id: "d-1", durationMs: 1000 },
  ];

  it("shows a signed percent and 'better' color when the field decreased and down is better", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells).toHaveLength(3);
    expect(cells[0].getAttribute("data-trend-kind")).toBe("changed");
    expect(cells[0].getAttribute("data-trend-direction")).toBe("down");
    expect(cells[0].getAttribute("data-trend-polarity")).toBe("better");
    expect(cells[0].textContent).toContain("-10%");
    expect(cells[0].className).toContain("text-emerald-600");
    view.unmount();
  });

  it("renders '- 0' for a flat delta", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS });
    const flatCell = view.container.querySelectorAll('[data-testid="widget-trend-cell"]')[1];
    expect(flatCell.getAttribute("data-trend-kind")).toBe("flat");
    expect(flatCell.textContent).toContain("0");
    view.unmount();
  });

  it("renders '- 0' for the last row when no more data is available", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS });
    const lastCell = view.container.querySelectorAll('[data-testid="widget-trend-cell"]')[2];
    expect(lastCell.getAttribute("data-trend-kind")).toBe("no-baseline");
    expect(lastCell.textContent).toContain("0");
    view.unmount();
  });

  it("renders '...' on the last row when more data is still loading", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS, hasMore: true });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells[cells.length - 1].getAttribute("data-trend-kind")).toBe("pending");
    expect(cells[cells.length - 1].textContent).toContain("...");
    view.unmount();
  });

  it("re-evaluates a CEL field on the row below (same aggregation on both rows)", () => {
    const view = renderTrend({
      render: {
        kind: "table",
        columns: [
          { field: "id" },
          {
            field: "{{ durationMs / 1000 }}",
            label: "Trend (s)",
            format: "trend",
            trendBetter: "down",
            trendDisplay: "value",
          },
        ],
      },
      rows: [
        { id: "row-a", durationMs: 4000 },
        { id: "row-b", durationMs: 5000 },
      ],
    });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells[0].getAttribute("data-trend-kind")).toBe("changed");
    expect(cells[0].textContent).toContain("-1");
    view.unmount();
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
