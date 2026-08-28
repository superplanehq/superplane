import { render, screen, waitFor, within } from "@testing-library/react";
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
    waiting: text.indexOf("Waiting for user review"),
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
    expect(within(dialog).getByText("Waiting for user review")).toBeInTheDocument();
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
    expect(within(dialog).queryByText("Waiting for user review")).not.toBeInTheDocument();
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
      4,
    );
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Draft: rework refund telemetry",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Add retry handling to webhook delivery",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open HTTP 500 /api/v1/factories/0644043d-564b-47e4-95b0-f5be415d0742",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-backlog-column")).getByRole("button", {
        name: "Open Show a clearer empty state on the billing page",
      }),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("work-order-card-score-wo-review-pay-842")).toHaveAttribute("aria-valuenow", "5");
    });
    expect(screen.getByTestId("work-order-card-score-wo-review-pay-844")).toHaveAttribute("aria-valuenow", "4");
    expect(screen.getByTestId("work-order-card-score-wo-review-pay-845")).toHaveAttribute("aria-valuenow", "3");
    expect(screen.queryByLabelText("Plan phase")).not.toBeInTheDocument();
    expect(within(screen.getByLabelText("Implement phase")).getAllByRole("button", { name: /^Open / })).toHaveLength(3);
    expect(
      within(screen.getByLabelText("Implement phase")).getByRole("button", {
        name: "Open Add refund reconciliation test",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Implement phase")).getByRole("button", {
        name: "Open Notify on status change after a reopen",
      }),
    ).toBeInTheDocument();
    const webhookCard = within(screen.getByLabelText("Implement phase"))
      .getByRole("button", {
        name: "Open Review the refund webhook schema change",
      })
      .closest("article") as HTMLElement;
    expect(within(webhookCard).getByLabelText("Running")).toBeInTheDocument();
    expect(within(webhookCard).queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(within(webhookCard).queryByText("Run failed")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-verify-column")).getByRole("button", {
        name: "Open Ship idempotent refund retries",
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("work-order-card-wo-failed-refunds")).getByText("Waiting for user review"),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("lines-verify-column")).getAllByRole("button", { name: /^Open / })).toHaveLength(
      1,
    );
    expect(
      within(screen.getByLabelText("Verify phase")).getByRole("button", {
        name: "Open Add refund reason enum to schema",
      }),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("lines-done-column")).getAllByRole("button", { name: /^Open / })).toHaveLength(4);
    const failedCard = within(screen.getByTestId("lines-done-column"))
      .getByRole("button", {
        name: "Open Fix refund dispatcher timeout loop",
      })
      .closest("article") as HTMLElement;
    expect(within(failedCard).getByLabelText("Failed")).toBeInTheDocument();
    expect(within(failedCard).getByText("Run failed")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-done-column")).getByRole("button", {
        name: "Open Send refund receipts after provider confirm",
      }),
    ).toBeInTheDocument();
    const rejectedCard = within(screen.getByTestId("lines-done-column"))
      .getByRole("button", { name: "Open Replace the refund batch exporter" })
      .closest("article") as HTMLElement;
    expect(within(rejectedCard).getByLabelText("Rejected")).toBeInTheDocument();
    const canceledCard = within(screen.getByTestId("lines-done-column"))
      .getByRole("button", { name: "Open Migrate refunds to the v2 provider API" })
      .closest("article") as HTMLElement;
    expect(within(canceledCard).getByLabelText("Canceled")).toBeInTheDocument();
    await user.click(card);

    const dialog = await screen.findByTestId("work-order-split-run");
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-open-work-order")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("link", { name: "Open work order" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Open work order" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-ingest")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-analyze")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-plan")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-score")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "plan.md" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("link", { name: /feature\/rf-103/ })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Automations" })).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-implement-0")).toBeInTheDocument();
    expect(within(dialog).queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(within(dialog).queryByText(/Users see duplicate refund/)).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-peek-dialog")).not.toBeInTheDocument();
  }, 15000);

  it("opens a manually created backlog card with person source", async () => {
    const user = userEvent.setup();
    const line = REFUND_FACTORY_LINES[0];

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    await user.click(
      await screen.findByRole("button", { name: "Open Draft: rework refund telemetry" }, { timeout: 8000 }),
    );

    const dialog = await screen.findByTestId("work-order-split-run");
    const source = within(dialog).getByTestId("split-run-source");
    expect(within(source).getByRole("img", { name: "Leonardo DiCaprio" })).toBeInTheDocument();
    expect(within(source).getByText("Created manually")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-description")).toHaveTextContent(
      "Let a user add emoji reactions on a work order itself (not only on comments).",
    );
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
    expect(within(dialog).queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).queryByText("Create plan")).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-implement-0")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    expect(
      within(screen.getByTestId("work-order-card-wo-failed-refunds")).getByText("Waiting for user review"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open Ship idempotent refund retries" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Waiting for user review" })).toBeInTheDocument();
    expect(within(dialog).getByRole("link", { name: "Review PR #6812" })).toBeInTheDocument();
    const waitingNote = within(dialog).getByTestId("split-run-attention-note");
    expect(within(waitingNote).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(waitingNote).getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Open full screen" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: /Update manually/ })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Fix refund dispatcher timeout loop" }));
    dialog = await screen.findByTestId("work-order-split-run");
    const failedNote = within(dialog).getByTestId("split-run-attention-note");
    expect(within(failedNote).getByRole("heading", { name: "This task is closed as failed" })).toBeInTheDocument();
    expect(within(failedNote).queryByRole("link", { name: "Review the run" })).not.toBeInTheDocument();
    expect(within(failedNote).getByRole("button", { name: "Reopen" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Add refund reason enum to schema" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(await within(dialog).findByTestId("split-run-phase-checks-verify-1")).toBeInTheDocument();
    expect(within(dialog).getByText("Risk score")).toBeInTheDocument();
    expect(within(dialog).getByText("Code quality")).toBeInTheDocument();
    expect(within(dialog).getByText("Verify")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Add retry handling to webhook delivery" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Plan" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Ticket" })).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-overview-checks")).toHaveTextContent("Confidence score");
    await user.click(within(dialog).getByRole("tab", { name: "Automations" }));
    expect(within(dialog).queryByTestId("split-run-phase-ingest")).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(screen.queryByTestId("review-candidate-modal")).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));

    await user.click(screen.getByRole("button", { name: "Open Send refund receipts after provider confirm" }));
    dialog = await screen.findByTestId("work-order-split-run");
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("tab", { name: "Automations" }));
    expect(await within(dialog).findByTestId("split-run-phase-checks-verify-1")).toBeInTheDocument();
    expect(within(dialog).getByText("Risk score")).toBeInTheDocument();
    expect(within(dialog).getByText("Code quality")).toBeInTheDocument();
    expect(within(dialog).queryByRole("link", { name: /#510/ })).not.toBeInTheDocument();
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
