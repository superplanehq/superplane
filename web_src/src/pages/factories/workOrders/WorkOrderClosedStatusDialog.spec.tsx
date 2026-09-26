import { render, screen } from "@testing-library/react";
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

  it("loads failed tasks and offers Send to backlog on each row", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [{ id: "wo-failed", number: "106", title: "Fix refund dispatcher timeout loop", key: "RF-106" }],
      isLoading: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage: vi.fn(),
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          status="failed"
          organizationId="org-1"
          factoryId="factory-1"
          factoryKey="RF"
          lineId="line-1"
          onOpenChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(useFactoryWorkOrdersPage).toHaveBeenCalledWith("org-1", "factory-1", ["STATE_CLOSED"], 20, {
      results: ["RESULT_FAILED"],
      lineId: "line-1",
    });
    expect(screen.getByTestId("work-order-failed-dialog")).toHaveTextContent("These tasks closed as failed.");
    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    await user.click(screen.getByTestId("send-to-backlog-wo-failed"));
    expect(screen.getByTestId("send-work-order-to-backlog-form")).toBeInTheDocument();
  });

  it("explains Rejected and shows the empty state", () => {
    useFactoryWorkOrdersPage.mockReturnValue({
      orders: [],
      isLoading: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage: vi.fn(),
    });

    render(
      <MemoryRouter>
        <WorkOrderClosedStatusDialog
          open
          status="rejected"
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
    expect(screen.getByTestId("work-order-rejected-dialog")).toHaveTextContent(
      "Archive, Reject, and Stop and Close mark a task as Rejected.",
    );
    expect(screen.getByText("No rejected tasks.")).toBeInTheDocument();
  });
});
