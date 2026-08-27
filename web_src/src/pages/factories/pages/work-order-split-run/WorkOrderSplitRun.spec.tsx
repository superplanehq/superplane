import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import {
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_NOTIFY_ID,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER, BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import {
  LINE_BOARD_DONE_RECEIPTS_ORDER,
  LINE_BOARD_VERIFY_ENUM_ORDER,
} from "../../__fixtures__/lineMetricsFactoriesFixture";
import { OPEN_WORK_ORDER_CHECKS, VERIFY_STEP_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { buildSplitRunFooter } from "./splitRunFooter";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(props: ComponentProps<typeof WorkOrderSplitRunPopup>) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup {...props} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderSplitRun() {
  return renderPopup({ fixture: SPLIT_RUN_RUNNING });
}

async function openLogTab(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("tab", { name: "Automations" }));
}

describe("WorkOrderSplitRunPopup", () => {
  it("puts an Open automation run link on the Automations heading", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    const expand = screen.getByRole("link", { name: "Open automation run" });
    expect(expand).toHaveAttribute("data-testid", "split-run-log-expand");
    expect(expand).toHaveAttribute(
      "href",
      getWorkOrderRunHref(
        FACTORIES_ORGANIZATION_ID,
        PRIMARY_FACTORY_KEY,
        "app-refund-implementer",
        LINE_RUN_IMPLEMENT_NOTIFY_ID,
        { orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number },
      ),
    );
  });

  it("hides the Open automation run link when the popup has no factory", () => {
    renderSplitRun();

    expect(screen.queryByRole("link", { name: "Open automation run" })).not.toBeInTheDocument();
  });

  it("does not put an Open work order link next to close", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
    });

    expect(screen.queryByTestId("split-run-open-work-order")).not.toBeInTheDocument();

    expect(screen.queryByRole("link", { name: "Open work order" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("collapses finished steps and expands the running component stream", () => {
    renderSplitRun();

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className).toContain("w-[min(56rem,calc(100vw-5rem))]");
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Description" })).toBeInTheDocument();
    const runningDot = within(dialog).getByTestId("split-run-log-tab-dot");
    expect(runningDot).toHaveAttribute("title", "Running");
    expect(runningDot.querySelector(".animate-ping")).toBeTruthy();
    expect(within(dialog).getByTestId("split-run-header-actions")).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Automations" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("region", { name: "Run" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();

    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByText("Backlog")).toBeInTheDocument();
    expect(within(backlog).queryByTestId("split-run-phase-duration-backlog")).not.toBeInTheDocument();
    expect(within(backlog).getByRole("button", { name: "description.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();

    const plan = screen.getByTestId("split-run-phase-plan");
    expect(within(plan).getAllByText(/Create plan/).length).toBeGreaterThan(0);
    expect(within(plan).queryByTestId("split-run-phase-duration-plan")).not.toBeInTheDocument();
    expect(within(plan).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-plan")).not.toBeInTheDocument();

    const implement = screen.getByTestId("split-run-phase-implement");
    expect(within(implement).getAllByText(/Implementation/).length).toBeGreaterThan(0);
    expect(within(implement).queryByTestId(/^split-run-phase-duration-/)).not.toBeInTheDocument();
    expect(within(implement).getAllByRole("link", { name: /feature\/refund-retry/ }).length).toBeGreaterThan(0);
    expect(screen.getByTestId("split-run-stream-implement")).toBeInTheDocument();
    expect(within(implement).queryByText("Started")).not.toBeInTheDocument();
    expect(within(implement).getAllByText("Create Branch").length).toBeGreaterThan(0);
    expect(
      within(screen.getByTestId("split-run-stream-line-create-branch")).queryByText("Run Bash"),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-line-create-branch")).not.toHaveTextContent(/\d{2}:\d{2}:\d{2}/);
    expect(within(screen.getByTestId("split-run-stream-line-create-branch")).queryByText(">")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-line-create-branch")).queryByText("├──"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-line-create-branch")).queryByText("└──"),
    ).not.toBeInTheDocument();
    const implementStream = screen.getByTestId("split-run-stream-implement");
    const note = within(implementStream).getByText("Provide Plan");
    expect(note.closest("li")).not.toHaveTextContent("├──");
    expect(note.closest("li")).not.toHaveTextContent("└──");
    expect(note.closest("li")).toHaveTextContent("bash");
    expect(within(implementStream).getByText("Set Up Environment")).toBeInTheDocument();
    expect(within(implementStream).getAllByText("✓").length).toBeGreaterThan(0);
    expect(within(implementStream).getByText(/superplaneagent@superplane.com/)).toBeInTheDocument();
    expect(
      within(implementStream).getByText(
        "Now let's look at the messages file, factory_notification_consumer.go, and other referenced files.",
      ),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-implement")).queryByText("├──")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-implement")).queryByText("└──")).not.toBeInTheDocument();
    expect(within(implement).queryByText("did not run")).not.toBeInTheDocument();

    expect(screen.getAllByText("Implementation").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
  });

  it("shows produced artifacts on the automation line", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    const plan = screen.getByTestId("split-run-phase-plan");
    const planToggle = within(plan).getByRole("button", { name: /Create plan/ });
    const planArtifacts = within(plan).getByTestId("split-run-phase-artifacts-plan");
    expect(planToggle.parentElement).toBe(planArtifacts.parentElement);
    expect(within(planArtifacts).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-plan")).not.toBeInTheDocument();

    await user.click(planToggle);

    expect(within(plan).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-automation-header-plan")).getByRole("button", { name: "plan.md" }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-plan")).getByRole("button", { name: "plan.md" }),
    ).toBeInTheDocument();

    const implement = screen.getByTestId("split-run-phase-implement");
    const implementToggle = within(implement).getByRole("button", { name: /Implement/ });
    const implementArtifacts = within(implement).getByTestId("split-run-phase-artifacts-implement");
    expect(implementToggle.parentElement).toBe(implementArtifacts.parentElement);
    expect(within(implementArtifacts).getByRole("link", { name: /feature\/refund-retry/ })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-implement")).not.toBeInTheDocument();

    await user.click(implementToggle);

    expect(within(implement).getByTestId("split-run-phase-artifacts-implement")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-automation-header-implement")).getByRole("link", {
        name: /feature\/refund-retry/,
      }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-implement")).getByRole("link", { name: /feature\/refund-retry/ }),
    ).toBeInTheDocument();
  });

  it("shows canvas artifacts on a collapsed automation before it is opened", () => {
    renderSplitRun();

    const plan = screen.getByTestId("split-run-phase-plan");
    expect(screen.queryByTestId("split-run-stream-plan")).not.toBeInTheDocument();
    expect(within(plan).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(within(plan).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
  });

  it("highlights a log line when it is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-stream-line-create-branch")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-create-branch")).toHaveAttribute("data-highlighted", "true");
    expect(screen.queryByTestId("split-run-canvas-node-create-branch")).not.toBeInTheDocument();
  });

  it("puts risk score and code quality on the verify step", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-impl",
                  step: "Implement",
                  stepIndex: 0,
                  state: "STATE_FINISHED",
                  result: "RESULT_PASSED",
                },
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 1,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    const verifyChecks = screen.getByTestId("split-run-phase-checks-verify-1");
    const risk = within(verifyChecks).getByRole("button", { name: /Risk score/ });
    expect(risk).toHaveTextContent("Risk score");
    expect(risk.className).toContain("bg-amber-500/10");
    expect(risk.className).not.toContain("bg-red-700");
    expect(within(verifyChecks).getByText("Code quality")).toBeInTheDocument();
    expect(within(verifyChecks).queryByText("Test coverage")).not.toBeInTheDocument();
    expect(within(verifyChecks).queryByText("Confidence score")).not.toBeInTheDocument();
    expect(within(verifyChecks).queryByText("CI")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-header-actions")).toBeInTheDocument();
  });

  it("pins the pull request review to the waiting implement log", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Ship idempotent refund retries",
        lineDispatches: [
          {
            id: "dispatch-waiting",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    expect(screen.getByTestId("split-run-review")).toBeInTheDocument();
  });

  it("opens a compact check in the analysis dialog", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 2,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    await user.click(screen.getByTestId("split-run-check-check-risk-review"));

    expect(screen.getByRole("heading", { name: "Risk score" })).toBeInTheDocument();
    expect(screen.getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
  });

  it("keeps the state bar when logs are complete and the order waits with no note", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "dasdas",
        statusNotes: [],
        assignees: [{ id: "user-1", name: "test test" }],
        lineDispatches: [
          {
            id: "dispatch-wait",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-1",
                step: "dasdasdas",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    expect(screen.getByTestId("split-run-header-actions")).toBeInTheDocument();
    expect(screen.queryByText("This work order needs attention from test test.")).not.toBeInTheDocument();
  });

  it("keeps the running log visible when the work order has no note or checks", () => {
    renderPopup({ fixture: { ...SPLIT_RUN_RUNNING, waitingNotes: [], checks: [] } });

    expect(screen.getByTestId("split-run-header-actions")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Automations" })).toBeInTheDocument();
  });

  it("pins a review note and keeps Update manually off the note", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER) });

    const note = screen.getByTestId("split-run-attention-note");
    const header = screen.getByTestId("split-run-header-actions");
    expect(within(note).getByRole("heading", { name: "Listening for user review" })).toBeInTheDocument();
    expect(note).toHaveTextContent("This automation finished and opened PR #6812.");
    expect(note).toHaveTextContent("Tag @superplaneagent in comment to request changes.");
    expect(note).toHaveTextContent("Task will automatically close when the pull request is closed or merged.");
    expect(within(note).getByRole("link", { name: "Review PR #6812" })).toHaveAttribute(
      "href",
      "https://github.com/superplanehq/superplane/pull/6812",
    );
    expect(within(note).queryByText("PR Closure")).not.toBeInTheDocument();
    expect(within(note).queryByText(/ago/)).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: /Update manually/ })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(within(header).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(header).getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
  });

  it("hides Reject and Approve when the user cannot update the work order", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      canUpdate: false,
    });

    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stop")).not.toBeInTheDocument();
  });

  it("offers automation Stop on a live running work order", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-running",
      fixture: SPLIT_RUN_RUNNING,
    });

    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
  });

  it("hides automation Stop when the user cannot update the work order", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-running",
      fixture: SPLIT_RUN_RUNNING,
      canUpdate: false,
    });

    expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
  });

  it("hides automation Rerun when the failed phase has no step index", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-failed-implement",
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-failed-implement",
        title: "Later step",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-plan", step: "Planning", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
              { id: "e-impl", step: "Implement", state: "STATE_FINISHED", result: "RESULT_FAILED" },
              { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.queryByRole("button", { name: "Rerun" })).not.toBeInTheDocument();
  });

  it("pins a default failed note and keeps Reopen in the header", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_FAILED_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER),
    });

    const note = screen.getByTestId("split-run-attention-note");
    const header = screen.getByTestId("split-run-header-actions");
    expect(within(note).getByRole("heading", { name: "Implement did not pass" })).toBeInTheDocument();
    expect(within(note).getByText(/Open the run to see what failed/)).toBeInTheDocument();
    expect(within(note).getByRole("link", { name: "Review the run" })).toHaveAttribute(
      "href",
      expect.stringContaining(LINE_RUN_IMPLEMENT_FAILED_ID),
    );
    expect(within(note).getByRole("link", { name: "Review the run" })).toHaveAttribute(
      "href",
      expect.stringContaining("orderNumber=106"),
    );
    expect(within(note).queryByRole("button", { name: "Reopen" })).not.toBeInTheDocument();
    expect(within(header).getByRole("button", { name: "Reopen" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
  });

  it("offers Reject and Approve on a failed open implement", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-2",
        title: "Test work order 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              { id: "e-plan", step: "Planning", state: "STATE_FINISHED", result: "RESULT_PASSED" },
              {
                id: "e-impl",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
              },
            ],
          },
        ],
      }),
    });

    const header = screen.getByTestId("split-run-header-actions");
    expect(within(header).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(header).getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rerun step" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Choose how to stop" })).not.toBeInTheDocument();
  });

  it("puts Reject and Approve in the header on a running order", () => {
    renderSplitRun();

    const header = screen.getByTestId("split-run-header-actions");
    expect(within(header).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(header).getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stop")).not.toBeInTheDocument();
  });

  it("opens a draft on the description tab and keeps the log collapsed", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("split-run-work-order-tab").className).toContain("minmax(0,3fr)_minmax(0,2fr)");
    expect(screen.getByTestId("split-run-work-order-tab")).toBeInTheDocument();
    const pendingDot = screen.getByTestId("split-run-log-tab-dot");
    expect(pendingDot).toHaveAttribute("title", "Pending");
    expect(pendingDot.querySelector(".animate-ping")).toBeNull();
    const source = screen.getByTestId("split-run-source");
    expect(within(source).getByRole("img", { name: "Leonardo DiCaprio" })).toBeInTheDocument();
    expect(within(source).getByText("Created manually")).toBeInTheDocument();
    const header = screen.getByTestId("split-run-header-actions");
    const headerStart = within(header).getByRole("button", { name: "Start" });
    const headerReject = within(header).getByRole("button", { name: "Reject" });
    expect(headerReject.compareDocumentPosition(headerStart) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(
      within(screen.getByTestId("split-run-work-order-tab")).queryByRole("button", { name: "Start" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("This work order is a draft.")).not.toBeInTheDocument();
    expect(screen.queryByText("Start this work order")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-overview-sidebar")).queryByTestId("split-run-review"),
    ).not.toBeInTheDocument();
    await openLogTab(user);
    expect(screen.getByTestId("split-run-log-pane").className).not.toContain("minmax(0,3fr)_minmax(0,2fr)");
    expect(within(header).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(header).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByRole("button", { name: "Backlog" })).toHaveAttribute("aria-expanded", "false");
    expect(within(backlog).queryByText(/Created manually/)).not.toBeInTheDocument();
    expect(within(backlog).queryByText("Completed")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();
    expect(within(backlog).getAllByRole("button", { name: "description.md" }).length).toBeGreaterThan(0);
    expect(screen.queryByText("On Issue Label")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("puts artifacts on the right and check analyses under the description", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
    });

    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).getByTestId("split-run-description")).toHaveTextContent(
      "Webhook delivery stops after a transient provider error",
    );
    expect(within(tab).getByRole("button", { name: "Edit" })).toBeInTheDocument();
    const sidebar = within(tab).getByTestId("split-run-overview-sidebar");
    expect(within(sidebar).getByRole("heading", { name: "Source" })).toBeInTheDocument();
    expect(within(sidebar).getByText("GitHub issues")).toBeInTheDocument();
    expect(within(sidebar).getByRole("link", { name: "acme/payments-service#842" })).toHaveAttribute(
      "href",
      "https://github.com/acme/payments-service/issues/842",
    );
    expect(within(sidebar).getByRole("heading", { name: "Artifacts" })).toBeInTheDocument();
    expect(within(sidebar).getByText("plan.md")).toBeInTheDocument();
    expect(within(sidebar).queryByText("PAY-842")).not.toBeInTheDocument();
    expect(within(sidebar).queryByText("details.md")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-overview-checks")).toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-check-comment-wo-review-pay-842-confidence")).toHaveAttribute("open");
    expect(within(tab).getByTestId("split-run-check-comment-check-risk-review")).not.toHaveAttribute("open");
    expect(within(tab).getByTestId("split-run-check-comment-check-code-coverage")).not.toHaveAttribute("open");
    expect(within(tab).getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
    expect(within(tab).getByText(/The change replaces the retry policy/)).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-header-actions")).toBeInTheDocument();
  });

  it("scores a review draft by how suitable the GitHub issue is for an agent", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]) });

    const tab = screen.getByTestId("split-run-work-order-tab");
    const check = within(tab).getByTestId("split-run-check-comment-wo-review-pay-842-confidence");
    expect(check).toHaveAttribute("open");
    expect(within(check).getByText("Confidence score")).toBeInTheDocument();
    expect(within(check).getByText("High")).toBeInTheDocument();
    expect(within(check).getByText("This issue is a good fit for an agent on this factory line.")).toBeInTheDocument();
    expect(within(check).getByText(/The automation read this GitHub issue/)).toBeInTheDocument();
    expect(within(check).getByText(/how suitable the work is for an agent/)).toBeInTheDocument();
    expect(within(check).getByText(/retryable status codes and a hard attempt limit/)).toBeInTheDocument();
  });

  it("opens and closes a check with details and summary", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
    });

    const confidence = screen.getByTestId("split-run-check-comment-wo-review-pay-842-confidence");
    const risk = screen.getByTestId("split-run-check-comment-check-risk-review");
    const coverage = screen.getByTestId("split-run-check-comment-check-code-coverage");
    expect(confidence).toHaveAttribute("open");
    expect(risk).not.toHaveAttribute("open");
    expect(coverage).not.toHaveAttribute("open");

    await user.click(screen.getByTestId("split-run-check-comment-toggle-wo-review-pay-842-confidence"));
    expect(confidence).not.toHaveAttribute("open");

    await user.click(screen.getByTestId("split-run-check-comment-toggle-check-code-coverage"));
    expect(coverage).toHaveAttribute("open");
  });

  it("keeps description checks collapsed when the work order is not a draft", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 2,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    await user.click(screen.getByRole("tab", { name: "Description" }));
    expect(screen.getByTestId("split-run-check-comment-check-risk-review")).not.toHaveAttribute("open");
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
  });

  it("keeps the ingest confidence check on a downstream Description tab", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_ENUM_ORDER, { checks: VERIFY_STEP_CHECKS }),
    });

    await user.click(screen.getByRole("tab", { name: "Description" }));
    const tab = screen.getByTestId("split-run-work-order-tab");
    const check = within(tab).getByTestId(`split-run-check-comment-${LINE_BOARD_VERIFY_ENUM_ORDER.id}-confidence`);
    expect(check).not.toHaveAttribute("open");
    expect(within(check).getByText("Confidence score")).toBeInTheDocument();
    expect(within(check).getByText(/fit for an agent on this factory line/)).toBeInTheDocument();
    expect(within(tab).getByText("Risk score")).toBeInTheDocument();
  });

  it("shows the Ingest log when a GitHub automation created the draft", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER) });

    await openLogTab(user);
    expect(screen.getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-phase-backlog")).getAllByRole("button", { name: "description.md" }).length,
    ).toBeGreaterThan(0);
  });

  it("opens a done card on the description tab with Reopen in the header", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER) });

    expect(
      within(screen.getByTestId("split-run-header-actions")).getByRole("button", { name: "Reopen" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
  });

  it("hides invented files and ledger pull requests on a live work order", async () => {
    const user = userEvent.setup();
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: LINE_BOARD_DONE_RECEIPTS_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, { demoArtifacts: false }),
    });

    expect(screen.queryByRole("link", { name: /#510/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "closure.md" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /merge-screenshot/ })).not.toBeInTheDocument();

    await openLogTab(user);
    expect(screen.queryByRole("link", { name: /merge-screenshot/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /#510/ })).not.toBeInTheDocument();
  });

  it("shows a failed implement stream with Reject and Approve in the header", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(FAILED_WORK_ORDER) });

    expect(screen.getByTestId("split-run-header-actions")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-implement-0")).toBeInTheDocument();
  });

  it("expands the new implement run after a rerun of the same step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      {
        id: "wo-2",
        title: "Test work order 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              {
                id: "e-plan",
                step: "Planning",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                run: { id: "run-plan" },
              },
              {
                id: "e-impl-old",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                run: { id: "run-old" },
              },
              {
                id: "e-impl-new",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_STARTED",
                result: "RESULT_UNKNOWN",
                run: { id: "run-new" },
              },
            ],
          },
        ],
      },
      { lineId: "line-1" },
    );
    const implementPhases = fixture.phases.filter((phase) => phase.name === "Implement");

    renderPopup({ fixture });

    expect(screen.getByTestId(`split-run-phase-${implementPhases[1].id}`)).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId(`split-run-phase-${implementPhases[0].id}`)).not.toHaveAttribute("aria-current");
  });

  it("opens the selected step log when a log row is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-phase-plan")).getByRole("button", { name: /Create plan/ }));

    expect(screen.getByTestId("split-run-stream-plan")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-plan")).queryByText("Started")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-plan")).getAllByText("Create Implementation Plan").length).toBe(
      1,
    );
    const planStream = screen.getByTestId("split-run-stream-plan");
    expect(within(planStream).getByText("Clone Repo")).toBeInTheDocument();
    expect(within(planStream).getAllByText("✓").length).toBeGreaterThan(0);
    expect(within(planStream).getByText(/Cloning into/)).toBeInTheDocument();
    expect(within(planStream).getByText("Provide description")).toBeInTheDocument();
    expect(within(planStream).getByText("Write Implementation Plan")).toBeInTheDocument();
    expect(within(planStream).getByText("Use plan as output")).toBeInTheDocument();
    expect(within(planStream).getAllByText("bash").length).toBeGreaterThan(0);
    expect(within(planStream).getByText("prompt")).toBeInTheDocument();
    expect(within(planStream).getByText("Let me examine the key reference files in detail.")).toBeInTheDocument();
    expect(within(planStream).getByRole("button", { name: "Ran 2 commands" })).toBeInTheDocument();
    expect(within(planStream).queryByRole("button", { name: "Read 7 files, ran 35 commands" })).not.toBeInTheDocument();
    expect(within(planStream).queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
  });

  it("opens a mapped implement-running work order on the implement log", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Implement job",
        lineDispatches: [
          {
            id: "dispatch-1",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.getByRole("heading", { name: "Implement job" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-ingest")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-analyze")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-plan")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-score")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-implement-0")).toBeInTheDocument();
    expect(screen.getAllByText("Implementation").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-header-actions")).toBeInTheDocument();
  });

  it("highlights the PR Closure log for the selected canvas component", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        title: "Send refund receipts after provider confirm",
        lineStatus: "passed",
        footer: buildSplitRunFooter({ kind: "done" }),
        footerTone: "done",
        currentPhaseId: "done",
        phases: [
          ...SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" as const })),
          {
            id: "done",
            name: "Done",
            status: "passed",
            duration: "1m 12s",
            componentName: "PR Closure",
            artifacts: [],
            stream: [],
            canvasSteps: [],
          },
        ],
      },
    });

    await openLogTab(user);
    expect(screen.queryByTestId("split-run-stream-done")).not.toBeInTheDocument();
    await user.click(within(screen.getByTestId("split-run-phase-done")).getByRole("button", { name: /Done/ }));

    const stream = screen.getByTestId("split-run-stream-done");
    expect(within(stream).queryByText("Started")).not.toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /merge-screenshot/ })).toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /#510/ })).toBeInTheDocument();

    await user.click(within(screen.getByTestId("split-run-stream-line-find-work-order")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-find-work-order")).toHaveAttribute("data-highlighted", "true");
    expect(screen.queryByTestId("split-run-canvas-node-find-work-order")).not.toBeInTheDocument();
  });

  it("lets you rename the title and edit the description on a draft", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
    await user.click(screen.getByTestId("popup-work-order-title"));
    const titleInput = await screen.findByTestId("popup-work-order-title-input");
    await user.clear(titleInput);
    await user.type(titleInput, "Renamed draft");
    await user.keyboard("{Enter}");
    expect(screen.getByTestId("popup-work-order-title")).toHaveTextContent("Renamed draft");
  });

  it("does not let you edit a completed work order", () => {
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        footer: buildSplitRunFooter({ kind: "done" }),
        footerTone: "done",
      },
    });

    expect(screen.queryByTestId("popup-work-order-title")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
  });
});
