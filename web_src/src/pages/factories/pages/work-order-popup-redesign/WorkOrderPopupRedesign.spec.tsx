import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { TooltipProvider } from "@/ui/tooltip";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES, RUNNING_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
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
    expect(within(dialog).getByText("Backlog")).toBeInTheDocument();
    expect(within(dialog).getByText("Create plan")).toBeInTheDocument();
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
    expect(within(dialog).getByText("Backlog")).toBeInTheDocument();
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
    expect(within(screen.getByTestId("lines-backlog-column")).getAllByRole("button", { name: /^Open / })).toHaveLength(
      3,
    );
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Add retry handling to webhook delivery",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Return 409 when the invoice is already paid",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Show a clearer empty state on the billing page",
      }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("work-order-card-score-wo-review-pay-842")).toHaveAttribute("aria-valuenow", "5");
    expect(screen.getByTestId("work-order-card-score-wo-review-pay-844")).toHaveAttribute("aria-valuenow", "4");
    expect(screen.getByTestId("work-order-card-score-wo-review-pay-845")).toHaveAttribute("aria-valuenow", "3");
    expect(screen.queryByLabelText("Plan phase")).not.toBeInTheDocument();
    expect(within(screen.getByLabelText("Implement phase")).getAllByRole("button", { name: /^Open / })).toHaveLength(3);
    expect(
      within(screen.getByLabelText("Implement phase")).getByRole("button", {
        name: "Open Add refund reconciliation test",
      }),
    ).toBeInTheDocument();
    const failedCard = within(screen.getByLabelText("Implement phase"))
      .getByRole("button", {
        name: "Open Fix refund dispatcher timeout loop",
      })
      .closest("article") as HTMLElement;
    expect(within(failedCard).getByLabelText("Failed")).toBeInTheDocument();
    expect(within(failedCard).getByText("Run failed")).toBeInTheDocument();
    const webhookCard = within(screen.getByLabelText("Implement phase"))
      .getByRole("button", {
        name: "Open Review the refund webhook schema change",
      })
      .closest("article") as HTMLElement;
    expect(within(webhookCard).getByLabelText("Running")).toBeInTheDocument();
    expect(within(webhookCard).queryByText("Approval needed")).not.toBeInTheDocument();
    expect(within(webhookCard).queryByText("Run failed")).not.toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Verify phase")).getByRole("button", {
        name: "Open Ship idempotent refund retries",
      }),
    ).toBeInTheDocument();
    expect(within(screen.getByLabelText("Verify phase")).getAllByRole("button", { name: /^Open / })).toHaveLength(2);
    expect(
      within(screen.getByLabelText("Verify phase")).getByRole("button", {
        name: "Open Add refund reason enum to schema",
      }),
    ).toBeInTheDocument();
    expect(within(screen.getByLabelText("Done phase")).getAllByRole("button", { name: /^Open / })).toHaveLength(3);
    expect(
      within(screen.getByLabelText("Done phase")).getByRole("button", {
        name: "Open Send refund receipts after provider confirm",
      }),
    ).toBeInTheDocument();
    const rejectedCard = within(screen.getByLabelText("Done phase"))
      .getByRole("button", { name: "Open Replace the refund batch exporter" })
      .closest("article") as HTMLElement;
    expect(within(rejectedCard).getByLabelText("Rejected")).toBeInTheDocument();
    const canceledCard = within(screen.getByLabelText("Done phase"))
      .getByRole("button", { name: "Open Migrate refunds to the v2 provider API" })
      .closest("article") as HTMLElement;
    expect(within(canceledCard).getByLabelText("Canceled")).toBeInTheDocument();
    await user.click(card);

    const dialog = await screen.findByTestId("work-order-split-run");
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-open-work-order")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("link", { name: "Open work order" })).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-ingest")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-analyze")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-plan")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-score")).toBeInTheDocument();
    expect(within(dialog).getAllByRole("button", { name: "details.md" }).length).toBeGreaterThan(0);
    expect(within(dialog).getAllByRole("button", { name: "plan.md" }).length).toBeGreaterThan(0);
    expect(within(dialog).getByRole("link", { name: /feature\/rf-103/ })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-checks-score")).toHaveTextContent("4/5");
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-implement-0")).toBeInTheDocument();
    expect(within(dialog).queryByText("Review the pull request")).not.toBeInTheDocument();
    expect(within(dialog).queryByText(/Users see duplicate refund/)).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-peek-dialog")).not.toBeInTheDocument();
  }, 15000);

  it("matches popup content to the card state", async () => {
    const user = userEvent.setup();
    const line = REFUND_FACTORY_LINES[0];

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    await screen.findByRole("button", { name: "Open Add refund reconciliation test" }, { timeout: 8000 });

    await user.click(screen.getByRole("button", { name: "Open Add refund reconciliation test" }));
    let dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).queryByText("Review the pull request")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).getByText("Create plan")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Ship idempotent refund retries" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).getByText("Review the pull request")).toBeInTheDocument();
    expect(within(dialog).getByRole("link", { name: "Review PR #6812" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Fix refund dispatcher timeout loop" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).getByText("Implement did not pass")).toBeInTheDocument();
    expect(within(dialog).getByRole("link", { name: "Open failed run" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Add refund reason enum to schema" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).queryByText("Review the pull request")).not.toBeInTheDocument();
    expect(await within(dialog).findByTestId("split-run-checks")).toBeInTheDocument();
    expect(within(dialog).getByText("Verify")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));
    dialog = await screen.findByTestId("review-candidate-modal");
    expect(within(dialog).getByRole("heading", { name: "Add retry handling to webhook delivery" })).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Send refund receipts after provider confirm" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(await within(dialog).findByTestId("split-run-checks")).toBeInTheDocument();
    expect(within(dialog).getAllByRole("link", { name: /#510|Send refund receipts/ })[0]).toHaveAttribute(
      "target",
      "_blank",
    );
  }, 20000);

  it("dispatches a draft work order to the open line", async () => {
    const user = userEvent.setup();
    const line = REFUND_FACTORY_LINES[0];

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    const card = await screen.findByTestId("work-order-card-wo-review-pay-842", {}, { timeout: 8000 });
    await user.click(within(card).getByRole("button", { name: "Start" }));

    await waitFor(() => {
      expect(
        within(screen.getByTestId("lines-backlog-column")).queryByTestId("work-order-card-wo-review-pay-842"),
      ).not.toBeInTheDocument();
    });
    expect(
      within(screen.getByLabelText("Implement phase")).getByTestId("work-order-card-wo-review-pay-842"),
    ).toBeInTheDocument();
  }, 15000);
});
