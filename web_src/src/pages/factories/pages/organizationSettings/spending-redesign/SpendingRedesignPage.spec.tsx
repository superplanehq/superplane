import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { PRIMARY_FACTORY_ID, STORYBOOK_ME_USER_ID, STORYBOOK_ME_USER_NAME } from "../../../__fixtures__/factoryPageIds";
import { SpendingRedesignPage } from "./SpendingRedesignPage";
import { SPENDING_CATALOGS, SPENDING_CREDIT, SPENDING_LEDGER, SPENDING_REDESIGN_NOW } from "./spendingRedesignMocks";
import { EMPTY_SPENDING_FILTERS, formatSpendingRangeCaption, rangeForPreset } from "./spendingRedesignLib";

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

  it("shows organization spend totals and separate model and VM explorers", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: "Spending" })).toBeInTheDocument();
    expect(
      screen.getByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Model usage" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "VM usage" })).toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-spend")).toHaveTextContent("Estimated spend");
    expect(screen.getByTestId("spending-kpi-tokens")).toHaveTextContent("Tokens");
    expect(screen.getByTestId("spending-kpi-vm")).toHaveTextContent("VM time");
    expect(screen.getByTestId("spending-kpi-credit")).toHaveTextContent("$41.24");
    expect(screen.getByTestId("spending-period")).toHaveTextContent("Last 30 days");
    expect(screen.queryByRole("tab", { name: "Custom" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Custom range" })).not.toBeInTheDocument();
    expect(
      within(within(screen.getByTestId("spending-model-usage")).getByTestId("spending-model-breakdown")).getByText(
        "Semaphore",
      ),
    ).toBeInTheDocument();
    expect(
      within(within(screen.getByTestId("spending-vm-usage")).getByTestId("spending-vm-breakdown")).getByText(
        "Semaphore",
      ),
    ).toBeInTheDocument();
  });

  it("renders the chart and breakdown in separate cards", () => {
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    const chart = within(models).getByTestId("spending-model-chart");
    const table = within(models).getByTestId("spending-model-breakdown");

    expect(chart).not.toContainElement(table);
    expect(table).not.toContainElement(chart);
    expect(chart.className).toContain("rounded-lg");
    expect(table.className).toContain("rounded-lg");
  });

  it("renders the KPI summary below the range control and above the usage sections", () => {
    renderPage();

    const range = screen.getByTestId("spending-period");
    const kpi = screen.getByTestId("spending-kpi-spend");
    const models = screen.getByTestId("spending-model-usage");

    expect(range.compareDocumentPosition(kpi) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(kpi.compareDocumentPosition(models) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("opens presets and a calendar in one period picker", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("spending-period"));

    expect(screen.getByRole("radio", { name: "Last 30 days", checked: true })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Last 7 days" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Last 24 hours" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Last 12 months" })).toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: "Custom" })).not.toBeInTheDocument();
    expect(screen.getByTestId("spending-period-picker")).toBeInTheDocument();
  });

  it("keeps model filters and VM filters on their own explorers", () => {
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    const machines = screen.getByTestId("spending-vm-usage");

    expect(within(models).getByTestId("spending-model-filter-models")).toBeInTheDocument();
    expect(within(models).queryByTestId("spending-model-filter-machines")).not.toBeInTheDocument();
    expect(within(machines).getByTestId("spending-vm-filter-machines")).toBeInTheDocument();
    expect(within(machines).queryByTestId("spending-vm-filter-models")).not.toBeInTheDocument();
  });

  it("places a group-by dropdown after a separator in each usage explorer", () => {
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    const machines = screen.getByTestId("spending-vm-usage");
    const modelGroupBy = within(models).getByTestId("spending-model-group-by");
    const vmGroupBy = within(machines).getByTestId("spending-vm-group-by");

    expect(within(models).getByTestId("spending-model-filter-bar")).toContainElement(modelGroupBy);
    expect(within(machines).getByTestId("spending-vm-filter-bar")).toContainElement(vmGroupBy);
    expect(modelGroupBy).toHaveTextContent("Group by Workspaces");
    expect(vmGroupBy).toHaveTextContent("Group by Workspaces");
  });

  it("changes only the model table when a model group-by option is selected", async () => {
    const user = userEvent.setup();
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    await user.click(within(models).getByTestId("spending-model-group-by"));
    await user.click(screen.getByRole("menuitemradio", { name: "Models" }));

    expect(within(models).getByTestId("spending-model-group-by")).toHaveTextContent("Group by Models");
    expect(within(within(models).getByTestId("spending-model-breakdown")).getByText("Model")).toBeInTheDocument();
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).getByText("claude-sonnet-4-6"),
    ).toBeInTheDocument();
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).getByText("claude-opus-4-6"),
    ).toBeInTheDocument();
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).queryByText("sonnet"),
    ).not.toBeInTheDocument();
    expect(within(within(models).getByTestId("spending-model-breakdown")).queryByText("opus")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("spending-vm-usage")).getByTestId("spending-vm-group-by")).toHaveTextContent(
      "Group by Workspaces",
    );
  });

  it("offers model grouping only on model usage", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(within(screen.getByTestId("spending-model-usage")).getByTestId("spending-model-group-by"));
    expect(screen.getByRole("menuitemradio", { name: "Models" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitemradio", { name: "Machine types" })).not.toBeInTheDocument();
  });

  it("offers machine-type grouping only on VM usage", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(within(screen.getByTestId("spending-vm-usage")).getByTestId("spending-vm-group-by"));
    expect(screen.getByRole("menuitemradio", { name: "Machine types" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitemradio", { name: "Models" })).not.toBeInTheDocument();
  });

  it("narrows totals when the last 7 days are selected", async () => {
    const user = userEvent.setup();
    renderPage();

    const monthSpend = screen.getByTestId("spending-kpi-spend").textContent;
    await user.click(screen.getByTestId("spending-period"));
    await user.click(screen.getByRole("radio", { name: "Last 7 days" }));
    expect(screen.getByTestId("spending-period")).toHaveTextContent("Last 7 days");
    expect(screen.getByTestId("spending-kpi-spend").textContent).not.toBe(monthSpend);
  });

  it("keeps organization KPI totals when a workspace filter is applied", async () => {
    const user = userEvent.setup();
    renderPage();

    const spend = screen.getByTestId("spending-kpi-spend").textContent;
    const tokens = screen.getByTestId("spending-kpi-tokens").textContent;
    const models = screen.getByTestId("spending-model-usage");
    await user.click(within(models).getByTestId("spending-model-filter-workspaces"));
    await user.click(screen.getByRole("menuitemradio", { name: "Semaphore" }));

    expect(screen.getByTestId("spending-kpi-spend").textContent).toBe(spend);
    expect(screen.getByTestId("spending-kpi-tokens").textContent).toBe(tokens);
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).queryByText("Acme onboarding"),
    ).not.toBeInTheDocument();
  });

  it("shows the versioned model id in the model filter", async () => {
    const user = userEvent.setup();
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    await user.click(within(models).getByTestId("spending-model-filter-models"));
    await user.click(screen.getByRole("menuitemradio", { name: "claude-sonnet-4-6" }));

    expect(within(models).getByTestId("spending-model-filter-models")).toHaveTextContent("claude-sonnet-4-6");
    expect(within(models).getByTestId("spending-model-filter-models").textContent).not.toBe("sonnet");
  });

  it("replaces a filter selection instead of adding a second value", async () => {
    const user = userEvent.setup();
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    await user.click(within(models).getByTestId("spending-model-filter-workspaces"));
    await user.click(screen.getByRole("menuitemradio", { name: "Semaphore" }));
    expect(within(models).getByTestId("spending-model-filter-workspaces")).toHaveTextContent("Semaphore");

    await user.click(within(models).getByTestId("spending-model-filter-workspaces"));
    await user.click(screen.getByRole("menuitemradio", { name: "Acme onboarding" }));
    expect(within(models).getByTestId("spending-model-filter-workspaces")).toHaveTextContent("Acme onboarding");
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).queryByText("Semaphore"),
    ).not.toBeInTheDocument();
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).getByText("Acme onboarding"),
    ).toBeInTheDocument();
  });

  it("filters by workspace and can clear the filter", async () => {
    const user = userEvent.setup();
    renderPage();

    const models = screen.getByTestId("spending-model-usage");
    await user.click(within(models).getByTestId("spending-model-filter-workspaces"));
    await user.click(screen.getByRole("menuitemradio", { name: "Semaphore" }));
    expect(within(models).getByTestId("spending-model-filter-workspaces")).toHaveTextContent("Semaphore");
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).queryByText("Acme onboarding"),
    ).not.toBeInTheDocument();
    expect(within(within(models).getByTestId("spending-model-breakdown")).getByText("Semaphore")).toBeInTheDocument();

    await user.click(within(models).getByTestId("spending-model-clear-filters"));
    expect(within(models).getByTestId("spending-model-filter-workspaces")).toHaveTextContent("All workspaces");
    expect(
      within(within(models).getByTestId("spending-model-breakdown")).getByText("Acme onboarding"),
    ).toBeInTheDocument();
  });

  it("shows spend in dollars in both usage tables", () => {
    renderPage();

    const modelTable = within(screen.getByTestId("spending-model-usage")).getByTestId("spending-model-breakdown");
    const vmTable = within(screen.getByTestId("spending-vm-usage")).getByTestId("spending-vm-breakdown");

    expect(within(modelTable).getByText("Spend")).toBeInTheDocument();
    expect(within(modelTable).queryByText("Tokens")).not.toBeInTheDocument();
    expect(within(vmTable).getByText("Spend")).toBeInTheDocument();
    expect(within(vmTable).queryByText("Time")).not.toBeInTheDocument();
  });

  it("shows an empty state when the ledger has no rows in range", () => {
    renderPage({
      events: [],
      initialPeriod: "day",
    });

    expect(screen.getAllByTestId("spending-empty").length).toBe(4);
    expect(screen.getAllByText("No model usage is recorded for this period.")).toHaveLength(2);
    expect(screen.getAllByText("No VM usage is recorded for this period.")).toHaveLength(2);
  });

  it("opens on a custom range and user breakdown when those initials are set", () => {
    renderPage({
      initialPeriod: "custom",
      initialModelBreakdown: "user",
      initialCustomRange: rangeForPreset("week", SPENDING_REDESIGN_NOW),
      initialModelFilters: {
        ...EMPTY_SPENDING_FILTERS,
        userId: STORYBOOK_ME_USER_ID,
        workspaceId: PRIMARY_FACTORY_ID,
      },
    });

    const models = screen.getByTestId("spending-model-usage");
    expect(screen.getByTestId("spending-period")).toHaveTextContent(
      formatSpendingRangeCaption(rangeForPreset("week", SPENDING_REDESIGN_NOW)),
    );
    expect(within(models).getByTestId("spending-model-group-by")).toHaveTextContent("Group by Users");
    expect(within(models).getByTestId("spending-model-filter-users")).toHaveTextContent(STORYBOOK_ME_USER_NAME);
    expect(within(models).getByTestId("spending-model-filter-workspaces")).toHaveTextContent("Semaphore");
    expect(within(within(models).getByTestId("spending-model-breakdown")).getByText("User")).toBeInTheDocument();
  });
});
