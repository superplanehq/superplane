import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { PRIMARY_FACTORY_ID, STORYBOOK_ME_USER_ID } from "../../../__fixtures__/factoryPageIds";
import { SpendingRedesignPage } from "./SpendingRedesignPage";
import { SPENDING_CATALOGS, SPENDING_CREDIT, SPENDING_LEDGER, SPENDING_REDESIGN_NOW } from "./spendingRedesignMocks";
import { EMPTY_SPENDING_FILTERS, rangeForPreset } from "./spendingRedesignLib";

function renderPage(props?: Partial<ComponentProps<typeof SpendingRedesignPage>>) {
  return render(
    <ThemeProvider>
      <TooltipProvider>
        <SpendingRedesignPage
          catalogs={SPENDING_CATALOGS}
          credit={SPENDING_CREDIT}
          events={SPENDING_LEDGER}
          now={SPENDING_REDESIGN_NOW}
          {...props}
        />
      </TooltipProvider>
    </ThemeProvider>,
  );
}

describe("SpendingRedesignPage", () => {
  beforeEach(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => undefined;
    Element.prototype.releasePointerCapture ??= () => undefined;
    Element.prototype.scrollIntoView ??= () => undefined;
  });
  it("shows organization spend totals for the last 30 days", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: "Spending" })).toBeInTheDocument();
    expect(
      screen.getByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-spend")).toHaveTextContent("Estimated spend");
    expect(screen.getByTestId("spending-kpi-tokens")).toHaveTextContent("Tokens");
    expect(screen.getByTestId("spending-kpi-vm")).toHaveTextContent("VM time");
    expect(screen.getByTestId("spending-kpi-credit")).toHaveTextContent("$41.24");
    expect(screen.getByRole("tab", { name: "Month", selected: true })).toBeInTheDocument();
    expect(screen.getByText("Semaphore")).toBeInTheDocument();
  });

  it("narrows totals when the Week range is selected", async () => {
    const user = userEvent.setup();
    renderPage();

    const monthSpend = screen.getByTestId("spending-kpi-spend").textContent;
    await user.click(screen.getByRole("tab", { name: "Week" }));
    expect(screen.getByRole("tab", { name: "Week", selected: true })).toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-spend").textContent).not.toBe(monthSpend);
  });

  it("filters by workspace and can clear the filter", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("spending-filter-workspaces"));
    await user.click(screen.getByRole("menuitemcheckbox", { name: "Semaphore" }));
    expect(screen.getByTestId("spending-filter-workspaces")).toHaveTextContent("1 workspace");
    expect(screen.queryByText("Acme onboarding")).not.toBeInTheDocument();
    expect(screen.getByText("Semaphore")).toBeInTheDocument();

    await user.click(screen.getByTestId("spending-clear-filters"));
    expect(screen.getByTestId("spending-filter-workspaces")).toHaveTextContent("All workspaces");
    expect(screen.getByText("Acme onboarding")).toBeInTheDocument();
  });

  it("shows an empty state when the ledger has no rows in range", () => {
    renderPage({
      events: [],
      initialPeriod: "day",
    });

    expect(screen.getAllByTestId("spending-empty").length).toBeGreaterThan(0);
    expect(screen.getAllByText("No factory usage is recorded for this period.").length).toBeGreaterThan(0);
  });

  it("opens on a custom range and user breakdown when those initials are set", () => {
    renderPage({
      initialPeriod: "custom",
      initialBreakdown: "user",
      initialCustomRange: rangeForPreset("week", SPENDING_REDESIGN_NOW),
      initialFilters: {
        ...EMPTY_SPENDING_FILTERS,
        userIds: [STORYBOOK_ME_USER_ID],
        workspaceIds: [PRIMARY_FACTORY_ID],
      },
    });

    expect(screen.getByRole("tab", { name: "Custom", selected: true })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Users", selected: true })).toBeInTheDocument();
    expect(screen.getByTestId("spending-filter-users")).toHaveTextContent("1 user");
    expect(screen.getByTestId("spending-filter-workspaces")).toHaveTextContent("1 workspace");
    expect(within(screen.getByTestId("spending-breakdown")).getByText("User")).toBeInTheDocument();
  });
});
