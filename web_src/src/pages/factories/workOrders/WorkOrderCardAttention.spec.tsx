import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import { formatRelative } from "@/lib/datetime";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const organizationId = "org-1";
const factoryKey = "RF";
const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: factoryKey };

describe("WorkOrderCard attention", () => {
  const waitingOrder: FactoriesWorkOrder = {
    id: "wo-waiting",
    number: "12",
    title: "Ship refund retries",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    statusNotes: [{ key: "pr-closure", headline: "Waiting for user review", body: "Tag the agent." }],
    lineDispatches: [],
    assignees: [],
  };

  const cardProps = {
    organizationId,
    factoryKey,
    factoryLines: [{ id: "line-a", name: "hotfix" }] as FactoriesFactoryLine[],
    canDispatch: true,
    canAssign: true,
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
    onOpen: vi.fn(),
  };

  it("shows Waiting for user review when the task has a status note", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard entry={buildWorkOrderListEntry(waitingOrder, factory)} {...cardProps} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Waiting for user review")).toBeInTheDocument();
    expect(screen.queryByText("Addressing user feedback")).not.toBeInTheDocument();
    const pill = screen.getByText("Waiting for user review").closest("span[title]");
    expect(pill).toBeInstanceOf(HTMLElement);
    if (!(pill instanceof HTMLElement)) {
      throw new Error("expected a status pill");
    }
    expect(pill.className).toMatch(/rounded-full/);
    const time = screen.getByText(formatRelative(new Date(waitingOrder.updatedAt ?? "")));
    expect(time.parentElement).not.toContainElement(pill);
  });

  it("shows Waiting on status checks when a check wait is active", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={buildWorkOrderListEntry(waitingOrder, factory)}
            {...cardProps}
            waitingOnChecksOrderIds={new Set(["wo-waiting"])}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(screen.getByText("Waiting on status checks")).toBeInTheDocument();
    expect(screen.queryByText("Addressing user feedback")).not.toBeInTheDocument();
  });

  it("shows Status checks passed after a check wait finishes", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={buildWorkOrderListEntry(waitingOrder, factory)}
            {...cardProps}
            checksPassedOrderIds={new Set(["wo-waiting"])}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Waiting for user review")).toBeInTheDocument();
    expect(screen.getByText("Status checks passed")).toBeInTheDocument();
    expect(screen.queryByText("Waiting on status checks")).not.toBeInTheDocument();
  });

  it("shows Automatic fixes paused after the check handler hits the attempt limit", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={buildWorkOrderListEntry(waitingOrder, factory)}
            {...cardProps}
            fixesPausedOrderIds={new Set(["wo-waiting"])}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Automatic fixes paused")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(screen.queryByText("Status checks passed")).not.toBeInTheDocument();
  });

  it("shows Addressing user feedback when a PR-feedback run is active", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={buildWorkOrderListEntry(waitingOrder, factory)}
            {...cardProps}
            addressingFeedbackOrderIds={new Set(["wo-waiting"])}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Addressing user feedback")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
  });
});
