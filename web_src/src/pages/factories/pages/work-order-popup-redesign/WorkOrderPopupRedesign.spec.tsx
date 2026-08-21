import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { TooltipProvider } from "@/ui/tooltip";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { WorkOrderPopupRedesignPlayground } from "./WorkOrderPopupRedesignPlayground";
import { AGENT_WORK_POPUP_RUNNING } from "./workOrderPopupMocks";

function renderPlayground() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkOrderPopupRedesignPlayground initialConcept="job" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

function sectionOrder(dialog: HTMLElement) {
  const text = dialog.textContent ?? "";
  return {
    waiting: text.indexOf("Review the pull request"),
    scores: text.indexOf("Scores"),
    log: text.indexOf("Log"),
  };
}

describe("WorkOrderPopupRedesignPlayground", () => {
  it("opens the job report without ticket chrome or an artifact preview", () => {
    renderPlayground();

    const dialog = screen.getByTestId("work-order-popup-job");
    expect(dialog).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Reconcile duplicate refunds in ledger" })).toBeInTheDocument();
    expect(within(dialog).getByText("Review the pull request")).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Scores" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Outputs" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "description.md" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Artifacts" })).not.toBeInTheDocument();
    expect(within(dialog).queryByText(/Users see duplicate refund/)).not.toBeInTheDocument();
    expect(within(dialog).getByRole("link", { name: /#482|Fix duplicate refund/ })).toHaveAttribute("target", "_blank");
    expect(within(dialog).getByRole("link", { name: /feature\/refund-retry/ })).toHaveAttribute("target", "_blank");
    expect(within(dialog).getByText("plan-and-implement")).toBeInTheDocument();
    expect(within(dialog).getByText("Plan")).toBeInTheDocument();
    expect(within(dialog).getByText("Implement")).toBeInTheDocument();
    expect(within(dialog).getByText("Verify")).toBeInTheDocument();
    expect(within(dialog).getByText("Done")).toBeInTheDocument();
    expect(within(dialog).getAllByText("Completed").length).toBeGreaterThan(0);
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();

    const order = sectionOrder(dialog);
    expect(order.waiting).toBeGreaterThan(-1);
    expect(order.waiting).toBeLessThan(order.scores);
    expect(order.scores).toBeLessThan(order.log);
  });

  it("keeps the log on a running job and hides scores", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderPopupRedesignPlayground initialConcept="job" fixture={AGENT_WORK_POPUP_RUNNING} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const dialog = screen.getByTestId("work-order-popup-job");
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Scores" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).getByText("Implement")).toBeInTheDocument();
    expect(within(dialog).queryByText("Review the pull request")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Outputs" })).not.toBeInTheDocument();
  });

  it("opens a markdown output in a second popup", async () => {
    const user = userEvent.setup();
    renderPlayground();
    const dialog = screen.getByTestId("work-order-popup-job");

    await user.click(within(dialog).getByRole("button", { name: "description.md" }));

    expect(screen.getByRole("heading", { name: "description.md" })).toBeInTheDocument();
    expect(screen.getByText(/Users see duplicate refund/)).toBeInTheDocument();
  });
});

describe("Line board job popup", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens the job report from a line-board card", async () => {
    const user = userEvent.setup();
    const line = REFUND_FACTORY_LINES[0];

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    const card = await screen.findByRole("button", { name: "Open Add refund reconciliation test" }, { timeout: 8000 });
    expect(screen.getByLabelText("Backlog")).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-column")).toBeInTheDocument();
    expect(screen.getByLabelText("Backlog menu")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Draft: rework refund telemetry",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).queryByRole("button", {
        name: "Open Reconcile duplicate refunds in ledger",
      }),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Plan phase")).getByRole("button", {
        name: "Open Reconcile duplicate refunds in ledger",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Implement phase")).getByRole("button", {
        name: "Open Add refund reconciliation test",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Verify phase")).getByRole("button", {
        name: "Open Add refund reason enum to schema",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Done phase")).getByRole("button", { name: "Open Backfill refund audit trail" }),
    ).toBeInTheDocument();
    await user.click(card);

    const dialog = await screen.findByTestId("work-order-popup-job");
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Outputs" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "description.md" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Scores" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).getByText("Implement")).toBeInTheDocument();
    expect(within(dialog).queryByText("Review the pull request")).not.toBeInTheDocument();
    expect(within(dialog).queryByText(/Users see duplicate refund/)).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-peek-dialog")).not.toBeInTheDocument();
  }, 15000);
});
