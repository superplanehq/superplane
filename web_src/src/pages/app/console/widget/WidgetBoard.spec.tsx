import { render, screen, fireEvent, act, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetBoard } from "./WidgetBoard";
import type { WidgetBoardRender } from "./types";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
  component: "start",
};

function baseRender(overrides: Partial<WidgetBoardRender> = {}): WidgetBoardRender {
  return {
    kind: "board",
    groupBy: "status",
    lanes: [
      { value: "Todo" },
      { value: "In Progress", label: "In progress", color: "blue" },
      { value: "Done", color: "green" },
    ],
    card: { titleField: "title" },
    ...overrides,
  };
}

const ROWS = [
  { id: "row-1", title: "Fix onboarding", status: "Todo" },
  { id: "row-2", title: "Ship board panel", status: "in progress" },
  { id: "row-3", title: "Migrate memory", status: "DONE" },
  { id: "row-4", title: "Stray task", status: "Blocked" },
];

function renderBoard(
  props: Partial<React.ComponentProps<typeof WidgetBoard>> & { canRunNodes?: boolean } = {},
  nodes: SuperplaneComponentsNode[] = [START_NODE],
) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { canRunNodes = false, ...boardProps } = props;
  const mergedRender = boardProps.render ?? baseRender();
  const mergedRows = boardProps.rows ?? ROWS;
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={nodes}
          canRunNodes={canRunNodes}
          onTriggerNode={() => undefined}
        >
          <WidgetBoard
            {...boardProps}
            render={mergedRender}
            rows={mergedRows}
            isLoading={boardProps.isLoading ?? false}
          />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetBoard grouping", () => {
  it("groups rows into configured lanes case-insensitively", () => {
    renderBoard();
    const lanes = screen.getAllByTestId("widget-board-lane");
    expect(lanes).toHaveLength(3);
    expect(lanes[0].getAttribute("data-lane-key")).toBe("lane:Todo");
    expect(lanes[1].getAttribute("data-lane-key")).toBe("lane:In Progress");
    expect(lanes[2].getAttribute("data-lane-key")).toBe("lane:Done");
    expect(within(lanes[0]).getByText("Fix onboarding")).toBeTruthy();
    expect(within(lanes[1]).getByText("Ship board panel")).toBeTruthy();
    expect(within(lanes[2]).getByText("Migrate memory")).toBeTruthy();
  });

  it("hides unmatched rows by default", () => {
    renderBoard();
    expect(screen.queryByText("Stray task")).toBeNull();
  });

  it("renders the `Other` lane when configured", () => {
    renderBoard({ render: baseRender({ otherLane: true }) });
    const lanes = screen.getAllByTestId("widget-board-lane");
    expect(lanes).toHaveLength(4);
    expect(lanes[3].getAttribute("data-lane-key")).toBe("__other__");
    expect(within(lanes[3]).getByText("Stray task")).toBeTruthy();
  });

  it("shows lane counts in the header badge", () => {
    renderBoard({ render: baseRender({ otherLane: true }) });
    const badges = screen.getAllByTestId("widget-board-lane-count").map((el) => el.textContent);
    expect(badges).toEqual(["1", "1", "1", "1"]);
  });

  it("renders the empty message when no rows are present", () => {
    renderBoard({ rows: [], render: baseRender({ emptyMessage: "Nothing yet" }) });
    expect(screen.getByTestId("widget-board-empty").textContent).toBe("Nothing yet");
  });

  it("renders the empty message when no rows match a configured lane", () => {
    renderBoard({
      rows: [{ id: "stray", title: "Stray task", status: "Unknown" }],
      render: baseRender({ emptyMessage: "No matching tasks" }),
    });
    expect(screen.getByTestId("widget-board-empty").textContent).toBe("No matching tasks");
  });

  it("loads more progressive rows from the board footer", () => {
    const onLoadMore = vi.fn();
    renderBoard({ hasMore: true, onLoadMore });
    fireEvent.click(screen.getByTestId("widget-table-load-more-button"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });
});

describe("WidgetBoard sort", () => {
  const SORT_ROWS = [
    { id: "a", title: "First", status: "Todo", updatedAt: 100 },
    { id: "b", title: "Second", status: "Todo", updatedAt: 300 },
    { id: "c", title: "Third", status: "Todo", updatedAt: 200 },
  ];

  it("sorts cards within a lane by the configured order", () => {
    renderBoard({
      rows: SORT_ROWS,
      render: baseRender({ sort: { field: "updatedAt", order: "desc" } }),
    });
    const cards = within(screen.getAllByTestId("widget-board-lane")[0]).getAllByTestId("widget-board-card");
    const titles = cards.map((c) => within(c).getByText(/First|Second|Third/).textContent);
    expect(titles).toEqual(["Second", "Third", "First"]);
  });
});

describe("WidgetBoard card fields", () => {
  it("renders configured card fields with formatted values", () => {
    renderBoard({
      rows: [{ id: "row-1", title: "T1", status: "Todo", url: "https://example.test/1" }],
      render: baseRender({
        card: {
          titleField: "title",
          fields: [{ field: "url", format: "link", label: "URL" }],
        },
      }),
    });
    const cards = screen.getAllByTestId("widget-board-card");
    const link = within(cards[0]).getByRole("link");
    expect(link.getAttribute("href")).toBe("https://example.test/1");
    expect(link.getAttribute("target")).toBe("_blank");
  });
});

describe("WidgetBoard row actions", () => {
  const ACTION_ROWS = [{ id: "row-1", title: "T1", status: "Todo", service: "api" }];
  const RENDER: WidgetBoardRender = baseRender({
    rowActions: [
      {
        kind: "trigger",
        label: "Kickoff",
        node: "start",
        show: 'status == "Todo"',
      },
    ],
  });

  it("fires the trigger callback when clicked", async () => {
    const onTrigger = vi.fn().mockResolvedValue(undefined);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
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
            <WidgetBoard render={RENDER} rows={ACTION_ROWS} isLoading={false} />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const triggers = screen.getAllByTestId("widget-row-action-start");
    expect(triggers).toHaveLength(1);
    await act(async () => {
      fireEvent.click(triggers[0]);
    });
    expect(onTrigger).toHaveBeenCalledWith(
      "start-id",
      expect.objectContaining({ hookName: "run", successLabel: "Kickoff" }),
    );
  });

  it("renders the trigger disabled when canRunNodes is false", () => {
    renderBoard({ rows: ACTION_ROWS, render: RENDER });
    expect(screen.getByTestId("widget-row-action-start")).toBeDisabled();
  });

  it("hides non-manual-run triggers", () => {
    const PR_NODE: SuperplaneComponentsNode = {
      id: "pr-id",
      name: "on-pr",
      type: "TYPE_TRIGGER",
      component: "github.onPullRequest",
    };
    renderBoard(
      {
        rows: ACTION_ROWS,
        render: baseRender({
          rowActions: [{ kind: "trigger", label: "Reopen", node: "on-pr" }],
        }),
        canRunNodes: true,
      },
      [PR_NODE],
    );
    expect(screen.queryByTestId("widget-row-action-on-pr")).toBeNull();
  });
});
