import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "bun:test";

import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
} from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

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
  organizationId: "org-1",
  factoryKey: "RF",
  factoryLines: [{ id: "line-a", name: "hotfix" }] as FactoriesFactoryLine[],
  canDispatch: true,
  canAssign: true,
  dispatchingOrderIds: new Set<string>(),
  isAssigneesSaving: false,
  onDispatch: vi.fn(),
  onAssigneesSave: vi.fn(),
  onOpen: vi.fn(),
};

const experimentalFeatureHas = { current: (_id: string) => true };

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => experimentalFeatureHas.current(id),
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

const openPullRequest: FactoriesFactoryPullRequest = {
  id: "pr-2323",
  workOrderId: "wo-waiting",
  number: "2323",
  url: "https://github.com/acme/payments/pull/2323",
  title: "Ship refund retries",
  state: "STATE_OPEN",
};

function renderCard(props: Partial<ComponentProps<typeof WorkOrderCard>> = {}) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WorkOrderCard entry={buildWorkOrderListEntry(waitingOrder, factory)} {...cardProps} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderCard pull request pill", () => {
  it("shows Review plus the number and hides Waiting for user review", () => {
    renderCard({ pullRequests: [openPullRequest] });

    const pill = screen.getByRole("link", { name: "Review pull request #2323." });
    expect(pill).toHaveTextContent("Review #2323");
    expect(pill).toHaveAttribute("href", "https://github.com/acme/payments/pull/2323");
    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
  });

  it("shows a Mergeable pill next to an open Review pill", () => {
    renderCard({ pullRequests: [{ ...openPullRequest, mergeable: true }] });

    expect(screen.getByRole("link", { name: "Review pull request #2323." })).toBeInTheDocument();
    expect(screen.getByTestId("work-order-mergeable-chip")).toHaveTextContent("Mergeable");
  });

  it("keeps Waiting for user review when the pull request is closed", () => {
    renderCard({
      pullRequests: [{ ...openPullRequest, id: "pr-closed", state: "STATE_CLOSED" }],
    });

    expect(screen.getByRole("link", { name: "Closed pull request #2323." })).toBeInTheDocument();
    expect(screen.getByText("Waiting for user review")).toBeInTheDocument();
  });

  it("does not show an attention badge when idle waiting has a pull request", () => {
    const idleOrder = { ...waitingOrder, id: "wo-idle", statusNotes: [] };
    renderCard({
      entry: buildWorkOrderListEntry(idleOrder, factory),
      pullRequests: [{ ...openPullRequest, workOrderId: "wo-idle" }],
    });

    expect(screen.getByRole("link", { name: "Review pull request #2323." })).toBeInTheDocument();
    expect(screen.queryByText("Needs attention")).not.toBeInTheDocument();
  });

  it("shows the completed check-wait title after checks pass", () => {
    renderCard({
      checksPassedOrderIds: new Set(["wo-waiting"]),
      checksPassedLabels: new Map([
        ["wo-waiting", "Checks passed on [2e46445](https://github.com/acme/app/commit/2e46445)"],
      ]),
    });

    expect(screen.getByText(/Checks passed on/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "2e46445" })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/commit/2e46445",
    );
    expect(screen.queryByText("Status checks passed")).not.toBeInTheDocument();
  });

  it("shows the check-fix activity on the card while the fixer runs", () => {
    renderCard({
      addressingFeedbackOrderIds: new Set(["wo-waiting"]),
      addressingFeedbackLabels: new Map([
        ["wo-waiting", "Fixing failed checks on [d8b80c2](https://github.com/acme/app/commit/d8b80c2)"],
      ]),
    });

    expect(screen.getByText(/Fixing failed checks on/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "d8b80c2" })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/commit/d8b80c2",
    );
    expect(screen.queryByText("Addressing user feedback")).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
  });

  it("keeps the checks-passed title next to the pull request pill", () => {
    renderCard({
      pullRequests: [openPullRequest],
      checksPassedOrderIds: new Set(["wo-waiting"]),
      checksPassedLabels: new Map([
        ["wo-waiting", "Checks passed on [2e46445](https://github.com/acme/app/commit/2e46445)"],
      ]),
    });

    expect(screen.getByRole("link", { name: "Review pull request #2323." })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "2e46445" })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/commit/2e46445",
    );
    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
  });

  it("names extra attached pull requests with a count", () => {
    renderCard({
      pullRequests: [
        openPullRequest,
        {
          id: "pr-1801",
          workOrderId: "wo-waiting",
          number: "1801",
          url: "https://github.com/acme/payments/pull/1801",
          state: "STATE_CLOSED",
        },
      ],
    });

    expect(screen.getByRole("link", { name: "Review pull request #2323. 1 more pull request." })).toHaveTextContent(
      "Review #2323 +1",
    );
  });

  it("hides the Mergeable chip when the pull request merge flag is off", () => {
    experimentalFeatureHas.current = () => false;
    renderCard({ pullRequests: [{ ...openPullRequest, mergeable: true }] });

    expect(screen.queryByTestId("work-order-mergeable-chip")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Review pull request #2323." })).toBeInTheDocument();
  });

  it("shows Status checks passed when the pull request is mergeable but the flag is off", () => {
    experimentalFeatureHas.current = () => false;
    renderCard({
      pullRequests: [{ ...openPullRequest, mergeable: true }],
      checksPassedOrderIds: new Set(["wo-waiting"]),
      checksPassedLabels: new Map([
        ["wo-waiting", "Checks passed on [2e46445](https://github.com/acme/app/commit/2e46445)"],
      ]),
    });

    expect(screen.getByRole("link", { name: "Review pull request #2323." })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "2e46445" })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/commit/2e46445",
    );
    expect(screen.queryByTestId("work-order-mergeable-chip")).not.toBeInTheDocument();
  });
});
