import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  DRAFT_WORK_ORDER,
  FAILED_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import { LINE_BOARD_DONE_RECEIPTS_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
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
  await user.click(screen.getByRole("tab", { name: "Log" }));
}

describe("WorkOrderSplitRunPopup", () => {
  it("does not put an Open work order link next to close", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
    });

    expect(screen.queryByTestId("split-run-open-work-order")).not.toBeInTheDocument();

    expect(screen.queryByRole("link", { name: "Open work order" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("collapses finished steps and expands the running component stream", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Description" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-log-tab-dot")).toHaveAttribute("title", "Running");
    expect(within(dialog).queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).getByRole("region", { name: "Run" })).toBeInTheDocument();

    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByText("Backlog")).toBeInTheDocument();
    expect(within(backlog).getByText("2s")).toBeInTheDocument();
    expect(within(backlog).getByRole("button", { name: "description.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();

    const plan = screen.getByTestId("split-run-phase-plan");
    expect(within(plan).getAllByText(/Create plan/).length).toBeGreaterThan(0);
    expect(within(plan).getByText("1m 12s")).toBeInTheDocument();
    expect(within(plan).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-plan")).not.toBeInTheDocument();

    const implement = screen.getByTestId("split-run-phase-implement");
    expect(within(implement).getAllByText(/Implementation/).length).toBeGreaterThan(0);
    expect(within(implement).getByText("4m")).toBeInTheDocument();
    expect(within(implement).getAllByRole("link", { name: /feature\/refund-retry/ }).length).toBeGreaterThan(0);
    expect(screen.getByTestId("split-run-stream-implement")).toBeInTheDocument();
    expect(within(implement).queryByText("Started")).not.toBeInTheDocument();
    expect(within(implement).getAllByText("Create Branch").length).toBeGreaterThan(0);
    expect(within(screen.getByTestId("split-run-stream-line-create-branch")).getByText("Run Bash")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-line-create-branch")).getByText(">")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-line-create-branch")).getByTestId("split-run-node-indent"),
    ).toBeInTheDocument();
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
    expect(within(implementStream).queryByText(/superplaneagent@superplane.com/)).not.toBeInTheDocument();
    await user.click(within(implementStream).getByText("Set Up Git User"));
    expect(within(implementStream).getByText(/superplaneagent@superplane.com/)).toBeInTheDocument();
    expect(
      within(implementStream).queryByText(
        "Now let's look at the messages file, factory_notification_consumer.go, and other referenced files.",
      ),
    ).not.toBeInTheDocument();
    await user.click(within(implementStream).getByText("Implementation"));
    expect(
      within(implementStream).getByText(
        "Now let's look at the messages file, factory_notification_consumer.go, and other referenced files.",
      ),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-implement")).queryByText("├──")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-implement")).queryByText("└──")).not.toBeInTheDocument();
    expect(within(implement).queryByText("did not run")).not.toBeInTheDocument();

    expect(screen.getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
    expect(screen.getAllByText("Implementation").length).toBeGreaterThan(0);
    expect(within(screen.getByTestId("run-overlay-compact-canvas")).getByText("Create Branch")).toBeInTheDocument();
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
  });

  it("shows produced artifacts on the collapsed automation line", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    const plan = screen.getByTestId("split-run-phase-plan");
    const planToggle = within(plan).getByRole("button", { name: /Create plan/ });
    const planArtifacts = within(plan).getByTestId("split-run-phase-artifacts-plan");
    expect(planToggle.parentElement).toBe(planArtifacts.parentElement);
    expect(within(planArtifacts).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-plan")).not.toBeInTheDocument();

    await user.click(planToggle);

    expect(within(plan).queryByTestId("split-run-phase-artifacts-plan")).not.toBeInTheDocument();
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

    expect(within(implement).queryByTestId("split-run-phase-artifacts-implement")).not.toBeInTheDocument();
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

  it("selects the canvas component when a log line is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-stream-line-create-branch")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-create-branch")).toHaveAttribute("data-highlighted", "true");
    expect(screen.getByTestId("split-run-canvas-node-create-branch")).toHaveAttribute("data-selected", "true");
    expect(screen.getByTestId("split-run-canvas-node-onrun-implement")).not.toHaveAttribute("data-selected");
  });

  it("keeps Edit Automation in the canvas overflow menu when no edit href is set", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(screen.getByTestId("split-run-canvas-menu"));
    expect(await screen.findByTestId("split-run-canvas-edit")).toHaveTextContent("Edit Automation");
  });

  it("opens Edit Automation from the canvas overflow menu", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING, canvasEditHref: () => "/edit-implementation" });

    await user.click(screen.getByTestId("split-run-canvas-menu"));
    const edit = await screen.findByTestId("split-run-canvas-edit");
    expect(edit).toHaveTextContent("Edit Automation");
    expect(edit).toHaveAttribute("href", "/edit-implementation");
  });

  it("places expand before the overflow menu and opens the automation run", () => {
    renderPopup({
      fixture: SPLIT_RUN_RUNNING,
      canvasEditHref: () => "/edit-implementation",
      canvasExpandHref: () => "/split-run-implementation",
    });

    const expand = screen.getByTestId("split-run-canvas-expand");
    const menu = screen.getByTestId("split-run-canvas-menu");
    expect(expand.compareDocumentPosition(menu) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(expand).toHaveAttribute("href", "/split-run-implementation");
    expect(expand).toHaveAttribute("aria-label", "Open automation run");
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
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
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

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
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

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(screen.queryByText("This work order needs attention from test test.")).not.toBeInTheDocument();
  });

  it("keeps the running log visible when the work order has no note or checks", () => {
    renderPopup({ fixture: { ...SPLIT_RUN_RUNNING, waitingNotes: [], checks: [] } });

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Log" })).toBeInTheDocument();
  });

  it("opens a draft on the description tab and keeps the log collapsed", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("split-run-work-order-tab")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-log-tab-dot")).toHaveAttribute("title", "Pending");
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    await openLogTab(user);
    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByText(/Created manually/)).toBeInTheDocument();
    expect(within(backlog).getByRole("button", { name: /Backlog/ })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();
    expect(within(backlog).getAllByRole("button", { name: "description.md" }).length).toBeGreaterThan(0);
    expect(screen.queryByText("On Issue Label")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("puts description on the left and checks plus artifacts on the right", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
    });

    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).getByTestId("split-run-description")).toHaveTextContent(
      "Webhook delivery stops after a transient provider error",
    );
    const sidebar = within(tab).getByTestId("split-run-overview-sidebar");
    expect(within(sidebar).getByTestId("split-run-overview-checks")).toBeInTheDocument();
    expect(within(sidebar).getByText("plan.md")).toBeInTheDocument();
    expect(within(sidebar).getByRole("heading", { name: "Artifacts" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
  });

  it("shows the Ingest canvas when a GitHub automation created the draft", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER) });

    await openLogTab(user);
    expect(screen.getByText("Ingest")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-canvas-node-on-issue-labeled")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-canvas-node-on-issue-assigned")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-phase-backlog")).getAllByRole("button", { name: "description.md" }).length,
    ).toBeGreaterThan(0);
  });

  it("shows a completed log without a footer", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER) });

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Log" })).toHaveAttribute("data-state", "active");
  });

  it("shows a failed implement stream without a footer", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(FAILED_WORK_ORDER) });

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-implement-0")).toBeInTheDocument();
  });

  it("opens the selected step canvas when a log row is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-phase-plan")).getByRole("button", { name: /Create plan/ }));

    expect(screen.getByTestId("split-run-stream-plan")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-plan")).queryByText("Started")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-plan")).getAllByText("Create Implementation Plan").length).toBe(
      1,
    );
    expect(within(screen.getByTestId("split-run-stream-plan")).queryByText("Clone Repo")).not.toBeInTheDocument();

    await user.click(within(screen.getByTestId("split-run-stream-plan")).getByText("Agent - Plan for GH Issue"));
    const planStream = screen.getByTestId("split-run-stream-plan");
    expect(within(planStream).getByText("Clone Repo")).toBeInTheDocument();
    expect(within(planStream).getAllByText("✓").length).toBeGreaterThan(0);
    expect(within(planStream).queryByText(/Cloning into/)).not.toBeInTheDocument();
    await user.click(within(planStream).getByText("Clone Repo"));
    expect(within(planStream).getByText(/Cloning into/)).toBeInTheDocument();
    expect(within(planStream).getByText("Provide description")).toBeInTheDocument();
    expect(within(planStream).getByText("Write Implementation Plan")).toBeInTheDocument();
    expect(within(planStream).getByText("Use plan as output")).toBeInTheDocument();
    expect(within(planStream).getAllByText("bash").length).toBeGreaterThan(0);
    expect(within(planStream).getByText("prompt")).toBeInTheDocument();
    expect(within(planStream).queryByText("Let me examine the key reference files in detail.")).not.toBeInTheDocument();

    await user.click(within(planStream).getByText("Write Implementation Plan"));
    expect(within(planStream).getByText("Let me examine the key reference files in detail.")).toBeInTheDocument();
    expect(within(planStream).getByRole("button", { name: "Ran 2 commands" })).toBeInTheDocument();
    expect(within(planStream).queryByRole("button", { name: "Read 7 files, ran 35 commands" })).not.toBeInTheDocument();
    expect(within(planStream).queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();
    expect(screen.getByText("Planning")).toBeInTheDocument();
    expect(screen.getAllByText("From GH issue?").length).toBeGreaterThan(0);
  });

  it("opens a mapped implement-running work order on the implement canvas", () => {
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
    expect(screen.getAllByText("From GH issue?").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
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

    expect(screen.queryByTestId("split-run-stream-done")).not.toBeInTheDocument();
    await user.click(within(screen.getByTestId("split-run-phase-done")).getByRole("button", { name: /Done/ }));

    const stream = screen.getByTestId("split-run-stream-done");
    expect(within(stream).queryByText("Started")).not.toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /merge-screenshot/ })).toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /#510/ })).toBeInTheDocument();

    await user.click(within(screen.getByTestId("split-run-stream-line-find-work-order")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-find-work-order")).toHaveAttribute("data-highlighted", "true");
    expect(screen.getByTestId("split-run-canvas-node-find-work-order")).toHaveAttribute("data-selected", "true");
  });
});
