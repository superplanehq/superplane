import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { SidebarEvent } from "../types";
import { HistoryQueuePage } from "./HistoryQueuePage";

vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) => classes.filter(Boolean).join(" "),
}));

vi.mock("../CompactSidebarEventRow", () => ({
  CompactSidebarEventRow: ({ event }: { event: SidebarEvent }) => (
    <div data-testid="compact-sidebar-event-row">{event.title}</div>
  ),
}));

vi.mock("../SidebarEventItem", () => ({
  SidebarEventItem: ({ event }: { event: SidebarEvent }) => <div data-testid="sidebar-event-item">{event.title}</div>,
}));

const event = {
  id: "event-1",
  title: "Event received",
  state: "success",
  isOpen: false,
  kind: "trigger",
  receivedAt: new Date("2026-05-27T12:19:23Z"),
} satisfies SidebarEvent;

function renderPage(props: Partial<React.ComponentProps<typeof HistoryQueuePage>> = {}) {
  return render(
    <HistoryQueuePage
      page="history"
      events={[event]}
      openEventIds={new Set()}
      onToggleOpen={vi.fn()}
      hasMoreItems={false}
      loadingMoreItems={false}
      showMoreCount={0}
      onLoadMoreItems={vi.fn()}
      {...props}
    />,
  );
}

describe("HistoryQueuePage", () => {
  it("renders compact rows in bottom layout mode", () => {
    renderPage({ compact: true });

    expect(screen.getByTestId("compact-sidebar-event-row")).toHaveTextContent("Event received");
    expect(screen.queryByTestId("sidebar-event-item")).not.toBeInTheDocument();
    expect(screen.queryByText("Run History")).not.toBeInTheDocument();
  });

  it("renders legacy rows in sidebar layout mode", () => {
    renderPage({ compact: false });

    expect(screen.getByTestId("sidebar-event-item")).toHaveTextContent("Event received");
    expect(screen.queryByTestId("compact-sidebar-event-row")).not.toBeInTheDocument();
    expect(screen.getByText("Run History")).toBeInTheDocument();
  });

  it("renders a compact load-more row when more history is available", () => {
    const onLoadMoreItems = vi.fn();

    renderPage({
      compact: true,
      hasMoreItems: true,
      showMoreCount: 12,
      onLoadMoreItems,
    });

    const loadMoreButton = screen.getByRole("button", { name: "Show 10 more" });
    fireEvent.click(loadMoreButton);

    expect(onLoadMoreItems).toHaveBeenCalledTimes(1);
  });
});
