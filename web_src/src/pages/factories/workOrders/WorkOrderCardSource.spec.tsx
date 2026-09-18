import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import superplaneIcon from "@/assets/superplane.svg";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

const baseOrder: FactoriesWorkOrder = {
  id: "wo-1",
  number: "12",
  title: "Ship refund retries",
  state: "STATE_OPEN",
  createdAt: "2024-06-01T00:00:00Z",
  updatedAt: "2024-06-02T00:00:00Z",
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

function renderCard(order: FactoriesWorkOrder, props: Partial<ComponentProps<typeof WorkOrderCard>> = {}) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WorkOrderCard entry={buildWorkOrderListEntry(order, factory)} {...cardProps} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderCard source icon", () => {
  it("links a GitHub origin to the ticket URL", () => {
    renderCard({
      ...baseOrder,
      origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
    });

    const icon = screen.getByTestId("work-order-card-source-wo-1");
    expect(icon).toHaveAttribute("href", "https://github.com/acme/payments/issues/12");
    expect(icon).toHaveAttribute("target", "_blank");
    expect(icon).toHaveAttribute("rel", "noopener noreferrer");
    expect(icon).toHaveAttribute("aria-label", "GitHub issues acme/payments#12");
    expect(icon).toHaveAttribute("data-slot", "button");
    expect(icon).toHaveAttribute("data-size", "icon-xs");
    expect(icon.className).toContain("size-6");
    expect(icon.querySelector("img")?.className).toContain("size-3.5");
  });

  it("shows the Jira icon for a Jira origin URL", () => {
    renderCard({
      ...baseOrder,
      origin: { url: "https://acme.atlassian.net/browse/DEV-3", label: "DEV-3" },
    });

    const icon = screen.getByTestId("work-order-card-source-wo-1");
    expect(icon.querySelector("img")).toHaveAttribute("src", jiraIcon);
    expect(icon).toHaveAttribute("href", "https://acme.atlassian.net/browse/DEV-3");
    expect(icon).toHaveAttribute("aria-label", "Jira issues DEV-3");
  });

  it("shows a Sentry icon without a link when only automation is present", () => {
    renderCard({
      ...baseOrder,
      createdBy: { automation: { appId: "sentry-intake", appName: "Sentry" } },
    });

    const icon = screen.getByTestId("work-order-card-source-wo-1");
    expect(icon.tagName).toBe("SPAN");
    expect(icon).not.toHaveAttribute("href");
    expect(icon.querySelector("img")).toHaveAttribute("src", sentryIcon);
    expect(icon).toHaveAttribute("aria-label", "Sentry exceptions");
  });

  it("renders a SuperPlane source mark for a manual task", () => {
    renderCard({ ...baseOrder, state: "STATE_DRAFT" });

    const icon = screen.getByTestId("work-order-card-source-wo-1");
    expect(icon.tagName).toBe("SPAN");
    expect(icon).not.toHaveAttribute("href");
    expect(icon.querySelector("img")).toHaveAttribute("src", superplaneIcon);
    expect(icon).toHaveAttribute("aria-label", "Created manually");
  });

  it("names the creator on a manual source mark", () => {
    renderCard({
      ...baseOrder,
      createdBy: { user: { id: "user-1", name: "Ada Lovelace" } },
    });

    const icon = screen.getByTestId("work-order-card-source-wo-1");
    expect(icon.tagName).toBe("SPAN");
    expect(icon).toHaveAttribute("aria-label", "Created manually by Ada Lovelace");
  });

  it("does not open the task when the source icon is clicked", async () => {
    const onOpen = vi.fn();
    renderCard(
      {
        ...baseOrder,
        origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
      },
      { onOpen },
    );

    await userEvent.click(screen.getByTestId("work-order-card-source-wo-1"));

    expect(onOpen).not.toHaveBeenCalled();
  });

  it("places the source icon in the title row", () => {
    renderCard({
      ...baseOrder,
      id: "wo-draft",
      state: "STATE_DRAFT",
      origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
    });

    const source = screen.getByTestId("work-order-card-source-wo-draft");
    const title = screen.getByText("Ship refund retries");
    const titleRow = title.parentElement;
    expect(titleRow).not.toBeNull();
    expect(titleRow?.contains(source)).toBe(true);
  });
});
