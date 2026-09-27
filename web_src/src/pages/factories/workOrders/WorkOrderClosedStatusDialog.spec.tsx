import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { useFactoryWorkOrdersPage } = vi.hoisted(() => ({
  useFactoryWorkOrdersPage: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrdersPage: (...args: unknown[]) => useFactoryWorkOrdersPage(...args),
  useWorkOrderArtifacts: () => ({ data: [], isLoading: false }),
  useSendWorkOrderToBacklog: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { WorkOrderClosedStatusDialog } from "./WorkOrderClosedStatusDialog";

describe("WorkOrderClosedStatusDialog", () => {
  beforeEach(() => {
    useFactoryWorkOrdersPage.mockReset();
  });

  it("loads Rejected and Canceled together and shows a status badge on each row", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [
        {
          id: "wo-rejected",
          number: "107",
          title: "Archive stale invoice matcher",
          key: "RF-107",
          state: "STATE_CLOSED",
          result: "RESULT_REJECTED",
        },
        {
          id: "wo-cancelled",
          number: "108",
          title: "Stop and close stale intake retry",
          key: "RF-108",
          state: "STATE_CLOSED",
          result: "RESULT_UNSPECIFIED",
          lineDispatches: [{ id: "d-cancel", state: "STATE_FINISHED", result: "RESULT_CANCELLED" }],
        },
      ],
      isLoading: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage: vi.fn(),
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          organizationId="org-1"
          factoryId="factory-1"
          factoryKey="RF"
          lineId="line-1"
          onOpenChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(useFactoryWorkOrdersPage).toHaveBeenCalledWith("org-1", "factory-1", ["STATE_CLOSED"], 20, {
      results: ["RESULT_REJECTED"],
      lineId: "line-1",
    });
    expect(screen.getByTestId("work-order-closed-status-dialog")).toHaveTextContent("Rejected and Canceled tasks.");
    expect(screen.getByTestId("work-order-status-badge-rejected")).toHaveTextContent("Rejected");
    expect(screen.getByTestId("work-order-status-badge-cancelled")).toHaveTextContent("Canceled");
    await user.click(screen.getByTestId("send-to-backlog-wo-rejected"));
    expect(screen.getByTestId("send-work-order-to-backlog-form")).toBeInTheDocument();
  });

  it("filters the loaded list with search", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [
        {
          id: "wo-failed",
          number: "106",
          title: "Fix refund dispatcher timeout loop",
          key: "RF-106",
          state: "STATE_CLOSED",
          result: "RESULT_FAILED",
        },
        {
          id: "wo-rejected",
          number: "107",
          title: "Archive stale invoice matcher",
          key: "RF-107",
          state: "STATE_CLOSED",
          result: "RESULT_REJECTED",
        },
      ],
      isLoading: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage: vi.fn(),
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          organizationId="org-1"
          factoryId="factory-1"
          factoryKey="RF"
          onOpenChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.type(screen.getByTestId("work-order-closed-status-search"), "invoice");
    expect(screen.getByText("Archive stale invoice matcher")).toBeInTheDocument();
    expect(screen.queryByText("Fix refund dispatcher timeout loop")).not.toBeInTheDocument();

    await user.clear(screen.getByTestId("work-order-closed-status-search"));
    await user.type(screen.getByTestId("work-order-closed-status-search"), "no-such-task");
    expect(screen.getByText("No tasks match this search.")).toBeInTheDocument();
  });

  it("loads more pages when search has no match on the loaded list", async () => {
    const user = userEvent.setup();
    const fetchNextPage = vi.fn();
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [
        {
          id: "wo-failed",
          number: "106",
          title: "Fix refund dispatcher timeout loop",
          key: "RF-106",
          state: "STATE_CLOSED",
          result: "RESULT_FAILED",
        },
      ],
      isLoading: false,
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage,
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          organizationId="org-1"
          factoryId="factory-1"
          factoryKey="RF"
          onOpenChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.type(screen.getByTestId("work-order-closed-status-search"), "invoice");
    await waitFor(() => {
      expect(fetchNextPage).toHaveBeenCalled();
    });
    expect(screen.getByText("No matching tasks on this page.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Load more" })).toBeInTheDocument();
  });

  it("does not retry a failed next page and explains the error", async () => {
    const user = userEvent.setup();
    const fetchNextPage = vi.fn();
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [
        {
          id: "wo-failed",
          number: "106",
          title: "Fix refund dispatcher timeout loop",
          key: "RF-106",
          state: "STATE_CLOSED",
          result: "RESULT_FAILED",
        },
      ],
      isLoading: false,
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: true,
      fetchNextPage,
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          organizationId="org-1"
          factoryId="factory-1"
          factoryKey="RF"
          onOpenChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.type(screen.getByTestId("work-order-closed-status-search"), "invoice");
    expect(fetchNextPage).not.toHaveBeenCalled();
    expect(screen.getByText("SuperPlane could not load more tasks.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Load more" })).toBeInTheDocument();
  });

  it("explains the empty list", () => {
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [],
      isLoading: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage: vi.fn(),
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          organizationId="org-1"
          factoryId="factory-1"
          factoryKey="RF"
          onOpenChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(useFactoryWorkOrdersPage).toHaveBeenCalledWith("org-1", "factory-1", ["STATE_CLOSED"], 20, {
      results: ["RESULT_REJECTED"],
      lineId: undefined,
    });
    expect(screen.getByText("No closed tasks.")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-closed-status-search")).not.toBeInTheDocument();
  });
});
