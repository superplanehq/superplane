import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryIntake } from "@/api-client";

import { useWorkOrderListState } from "../../lib/useWorkOrderListState";
import { WorkOrdersHeader } from "./WorkOrdersHeader";

function HeaderHarness({
  onCreateWorkOrder,
  canCreate = true,
  intakes = [],
  showPullRequestMerge = false,
  onOpenStatusDialog,
}: {
  onCreateWorkOrder: () => void;
  canCreate?: boolean;
  intakes?: FactoriesFactoryIntake[];
  showPullRequestMerge?: boolean;
  onOpenStatusDialog?: (status: "failed" | "rejected") => void;
}) {
  const state = useWorkOrderListState("factory-1");
  return (
    <WorkOrdersHeader
      state={state}
      entries={[]}
      factoryLines={[]}
      intakes={intakes}
      onCreateWorkOrder={onCreateWorkOrder}
      canCreate={canCreate}
      permissionsLoading={false}
      showPullRequestMerge={showPullRequestMerge}
      onOpenStatusDialog={onOpenStatusDialog}
    />
  );
}

describe("WorkOrdersHeader", () => {
  it("opens the create dialog in place instead of changing the path", async () => {
    const user = userEvent.setup();
    const onCreateWorkOrder = vi.fn();

    render(<HeaderHarness onCreateWorkOrder={onCreateWorkOrder} />);

    await user.click(screen.getByTestId("work-order-list-create-button"));

    expect(onCreateWorkOrder).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("work-order-list-create-button")).not.toHaveAttribute("href");
  });

  it("lists Source after Line, with configured tools and Created manually", async () => {
    const user = userEvent.setup();
    render(
      <HeaderHarness
        onCreateWorkOrder={vi.fn()}
        intakes={[
          { id: "github-1", source: "SOURCE_GITHUB_ISSUES" },
          { id: "github-2", source: "SOURCE_GITHUB_ISSUES" },
          { id: "jira-1", source: "SOURCE_JIRA_ISSUES" },
        ]}
      />,
    );

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    expect(screen.getByTestId("work-orders-filter-lineIds")).toBeInTheDocument();
    const source = screen.getByTestId("work-orders-filter-sourceIds");
    expect(source).toHaveTextContent("Source");
    expect(
      screen.getByTestId("work-orders-filter-lineIds").compareDocumentPosition(source) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("lists Label after Status, with Review only when merge is off", async () => {
    const user = userEvent.setup();
    render(<HeaderHarness onCreateWorkOrder={vi.fn()} />);

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    const labels = screen.getByTestId("work-orders-filter-labels");
    expect(labels).toHaveTextContent("Label");
    expect(
      screen.getByTestId("work-orders-filter-statuses").compareDocumentPosition(labels) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      labels.compareDocumentPosition(screen.getByTestId("work-orders-filter-lineIds")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.hover(labels);
    expect(await screen.findByTestId("work-orders-filter-labels-review")).toHaveTextContent("Review");
    expect(screen.queryByTestId("work-orders-filter-labels-mergeable")).not.toBeInTheDocument();
  });

  it("lists Mergeable in Label when the merge pill is on", async () => {
    const user = userEvent.setup();
    render(<HeaderHarness onCreateWorkOrder={vi.fn()} showPullRequestMerge />);

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-labels"));
    expect(await screen.findByTestId("work-orders-filter-labels-review")).toHaveTextContent("Review");
    expect(screen.getByTestId("work-orders-filter-labels-mergeable")).toHaveTextContent("Mergeable");
  });

  it("still offers Source when the factory has no intakes", async () => {
    const user = userEvent.setup();
    render(<HeaderHarness onCreateWorkOrder={vi.fn()} />);

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    expect(screen.getByTestId("work-orders-filter-sourceIds")).toHaveTextContent("Source");
  });

  it("opens Failed and Rejected from Status instead of applying a board filter", async () => {
    const user = userEvent.setup();
    const onOpenStatusDialog = vi.fn();
    render(<HeaderHarness onCreateWorkOrder={vi.fn()} onOpenStatusDialog={onOpenStatusDialog} />);

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-statuses"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-statuses-failed"));

    expect(onOpenStatusDialog).toHaveBeenCalledWith("failed");
    expect(screen.queryByTestId("work-orders-filter-trigger")).toHaveAttribute("aria-expanded", "false");

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-statuses"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-statuses-rejected"));
    expect(onOpenStatusDialog).toHaveBeenCalledWith("rejected");
  });

  it("shows a removable Source chip for a selected tool", async () => {
    const user = userEvent.setup();
    window.localStorage.setItem(
      "sp:work-orders:filters:factory-chip",
      JSON.stringify({
        statuses: [],
        labels: [],
        lineIds: [],
        sourceIds: ["github-issues", "manual"],
        assigneeIds: [],
      }),
    );
    function ChipHarness() {
      const state = useWorkOrderListState("factory-chip");
      return (
        <WorkOrdersHeader
          state={state}
          entries={[]}
          factoryLines={[]}
          intakes={[{ id: "github-1", source: "SOURCE_GITHUB_ISSUES" }]}
          onCreateWorkOrder={vi.fn()}
          canCreate
          permissionsLoading={false}
        />
      );
    }

    render(<ChipHarness />);
    expect(screen.getByText("Source is GitHub issues")).toBeInTheDocument();
    expect(screen.getByText("Created manually")).toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-clear")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove filter Source is GitHub issues" }));
    expect(screen.queryByText("Source is GitHub issues")).not.toBeInTheDocument();
    expect(screen.getByText("Created manually")).toBeInTheDocument();
  });
});
