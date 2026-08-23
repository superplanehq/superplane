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
  RUNNING_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
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
    expect(within(plan).getByText(/Planning/)).toBeInTheDocument();
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
    const planToggle = within(plan).getByRole("button", { name: /Plan/ });
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
    renderPopup({ fixture: splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER) });

    const plan = screen.getByTestId("split-run-phase-plan-0");
    expect(screen.queryByTestId("split-run-stream-plan-0")).not.toBeInTheDocument();
    expect(within(plan).getByTestId("split-run-phase-artifacts-plan-0")).toBeInTheDocument();
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

  it("keeps Edit in the canvas overflow menu when no edit href is set", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(screen.getByTestId("split-run-canvas-menu"));
    expect(await screen.findByTestId("split-run-canvas-edit")).toHaveTextContent("Edit");
  });

  it("opens Edit from the canvas overflow menu", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING, canvasEditHref: () => "/edit-implementation" });

    await user.click(screen.getByTestId("split-run-canvas-menu"));
    const edit = await screen.findByTestId("split-run-canvas-edit");
    expect(edit).toHaveTextContent("Edit");
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

  it("keeps every check pill on the owner and cost row for verify", () => {
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
                { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
                {
                  id: "e-impl",
                  step: "Implement",
                  stepIndex: 1,
                  state: "STATE_FINISHED",
                  result: "RESULT_PASSED",
                },
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

    const meta = screen.getByTestId("popup-owner-time-cost");
    expect(within(meta).getByTestId("split-run-checks")).toBeInTheDocument();
    expect(within(meta).getByText("Risk review")).toBeInTheDocument();
    expect(within(meta).getByText("Test coverage")).toBeInTheDocument();
    expect(within(meta).getByText("Confidence score")).toBeInTheDocument();
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
              { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    const review = screen.getByTestId("split-run-review");
    expect(review).toHaveTextContent("Review the pull request");
    expect(review).toHaveTextContent("Next step");
    expect(within(review).getByRole("link", { name: "Review PR #6812" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
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

    expect(screen.getByRole("heading", { name: "Risk review" })).toBeInTheDocument();
    expect(screen.getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
  });

  it("asks the assignee for attention when logs are complete and the order waits", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
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
        },
        { detailHref: "/org-1/workspaces/RF/work-order/101" },
      ),
    });

    const review = screen.getByTestId("split-run-review");
    expect(review).toHaveTextContent("Needs attention");
    expect(review).toHaveTextContent("This work order needs attention from test test.");
    expect(within(review).getByRole("link", { name: "Open work order" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/work-order/101",
    );
  });

  it("hides the review strip when the work order has no note or checks", () => {
    renderPopup({ fixture: { ...SPLIT_RUN_RUNNING, waitingNotes: [], checks: [] } });

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
  });

  it("prompts a draft backlog card to start the next stage", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    const review = screen.getByTestId("split-run-review");
    expect(review).toHaveTextContent("Start the next stage");
    expect(within(review).getByRole("button", { name: "Dispatch" })).toBeDisabled();
    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByText(/Created manually/)).toBeInTheDocument();
    expect(within(backlog).getByText("Leonardo DiCaprio created this work order manually.")).toBeInTheDocument();
    expect(within(backlog).getAllByRole("button", { name: "description.md" }).length).toBeGreaterThan(0);
    expect(screen.queryByText("On Issue Label")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("shows the Ingest canvas when a GitHub automation created the draft", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER) });

    expect(screen.getByText("Ingest")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-canvas-node-on-issue-labeled")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-canvas-node-on-issue-assigned")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-phase-backlog")).getAllByRole("button", { name: "description.md" }).length,
    ).toBeGreaterThan(0);
  });

  it("dispatches the draft to the line from the next-step button", async () => {
    const user = userEvent.setup();
    const onDispatch = vi.fn().mockResolvedValue(undefined);

    renderPopup({
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
      canDispatch: true,
      onDispatch,
    });

    await user.click(within(screen.getByTestId("split-run-review")).getByRole("button", { name: "Dispatch" }));

    expect(onDispatch).toHaveBeenCalledTimes(1);
  });

  it("shows a failed implement diagnosis in the log footer", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(FAILED_WORK_ORDER) });

    const review = screen.getByTestId("split-run-review");
    expect(review).toHaveTextContent("Implement did not pass");
    expect(within(review).getByRole("link", { name: "Open failed run" })).toBeInTheDocument();
    expect(review.className).toContain("status-failed-bg");
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("opens the selected step canvas when a log row is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-phase-plan")).getByRole("button", { name: /Plan/ }));

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

  it("opens a mapped plan-running work order on the plan canvas", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Plan job",
        lineDispatches: [
          {
            id: "dispatch-1",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.getByRole("heading", { name: "Plan job" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-phase-backlog")).getByText(/Ingest/)).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-plan-0")).toBeInTheDocument();
    expect(screen.getByText("Planning")).toBeInTheDocument();
    expect(screen.getAllByText("From GH issue?").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("split-run-phase-implement")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("highlights the PR Closure log for the selected canvas component", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        title: "Send refund receipts after provider confirm",
        lineStatus: "passed",
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

    const stream = screen.getByTestId("split-run-stream-done");
    expect(within(stream).queryByText("Started")).not.toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /merge-screenshot/ })).toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /#510/ })).toBeInTheDocument();

    await user.click(within(screen.getByTestId("split-run-stream-line-find-work-order")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-find-work-order")).toHaveAttribute("data-highlighted", "true");
    expect(screen.getByTestId("split-run-canvas-node-find-work-order")).toHaveAttribute("data-selected", "true");
  });
});
