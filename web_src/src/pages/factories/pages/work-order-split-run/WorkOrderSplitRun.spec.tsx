import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, FAILED_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderSplitRun() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkOrderSplitRunPopup fixture={SPLIT_RUN_RUNNING} />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WorkOrderSplitRunPopup", () => {
  it("collapses finished steps and expands the running component stream", () => {
    renderSplitRun();

    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).getByRole("region", { name: "Run" })).toBeInTheDocument();

    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByText("Backlog")).toBeInTheDocument();
    expect(within(backlog).getByRole("button", { name: "description.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();

    const plan = screen.getByTestId("split-run-phase-plan");
    expect(within(plan).getByText(/Refund Planner/)).toBeInTheDocument();
    expect(within(plan).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-plan")).not.toBeInTheDocument();

    const implement = screen.getByTestId("split-run-phase-implement");
    expect(within(implement).getByText(/Refund Implementer/)).toBeInTheDocument();
    expect(within(implement).getAllByRole("link", { name: /feature\/refund-retry/ }).length).toBeGreaterThan(0);
    expect(screen.getByTestId("split-run-stream-implement")).toBeInTheDocument();
    expect(within(implement).queryByText("Started")).not.toBeInTheDocument();
    expect(within(implement).getAllByText("Create Branch").length).toBeGreaterThan(0);
    expect(within(implement).getByText("Reading plan.md.")).toBeInTheDocument();

    expect(screen.getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
    expect(screen.getByText("Implementation")).toBeInTheDocument();
    expect(within(screen.getByTestId("run-overlay-compact-canvas")).getByText("Create Branch")).toBeInTheDocument();
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
  });

  it("opens Edit from the canvas overflow menu", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup fixture={SPLIT_RUN_RUNNING} canvasEditHref={() => "/edit-implementation"} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("split-run-canvas-menu"));
    const edit = await screen.findByTestId("split-run-canvas-edit");
    expect(edit).toHaveTextContent("Edit");
    expect(edit).toHaveAttribute("href", "/edit-implementation");
  });

  it("places expand before the overflow menu and opens the automation run", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup fixture={SPLIT_RUN_RUNNING} canvasExpandHref={() => "/split-run-implementation"} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const expand = screen.getByTestId("split-run-canvas-expand");
    const menu = screen.getByTestId("split-run-canvas-menu");
    expect(expand.compareDocumentPosition(menu) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(expand).toHaveAttribute("href", "/split-run-implementation");
    expect(expand).toHaveAttribute("aria-label", "Open automation run");
  });

  it("keeps every check pill on the owner and cost row for verify", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup
            fixture={splitRunFixtureForWorkOrder({
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
                    { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
                  ],
                },
              ],
            })}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const meta = screen.getByTestId("popup-owner-time-cost");
    expect(within(meta).getByTestId("split-run-checks")).toBeInTheDocument();
    expect(within(meta).getByText("Risk review")).toBeInTheDocument();
    expect(within(meta).getByText("Test coverage")).toBeInTheDocument();
    expect(within(meta).getByText("Confidence score")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
  });

  it("pins the pull request review to the waiting implement log", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup
            fixture={splitRunFixtureForWorkOrder({
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
            })}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const review = screen.getByTestId("split-run-review");
    expect(review).toHaveTextContent("Review the pull request");
    expect(review).toHaveTextContent("Next step");
    expect(within(review).getByRole("link", { name: "Review PR #6812" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("opens a compact check in the analysis dialog", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup
            fixture={splitRunFixtureForWorkOrder({
              ...OPEN_WORK_ORDER,
              title: "Add refund reason enum to schema",
              lineDispatches: [
                {
                  id: "dispatch-verify",
                  line: { id: "line-1", name: "plan-and-implement" },
                  state: "STATE_ACTIVE",
                  stepExecutions: [
                    { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
                  ],
                },
              ],
            })}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("split-run-check-check-risk-review"));

    expect(screen.getByRole("heading", { name: "Risk review" })).toBeInTheDocument();
    expect(screen.getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
  });

  it("hides the review strip when the work order has no note or checks", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup fixture={{ ...SPLIT_RUN_RUNNING, waitingNotes: [], checks: [] }} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
  });

  it("prompts a draft backlog card to start the next stage", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const review = screen.getByTestId("split-run-review");
    expect(review).toHaveTextContent("Start the next stage");
    expect(within(review).getByRole("link", { name: "Start Plan" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("shows a failed implement diagnosis in the log footer", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup fixture={splitRunFixtureForWorkOrder(FAILED_WORK_ORDER)} />
        </TooltipProvider>
      </MemoryRouter>,
    );

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
    expect(
      within(screen.getByTestId("split-run-stream-plan")).getByText("Reading the work order description."),
    ).toBeInTheDocument();
    expect(screen.getByText("Planning")).toBeInTheDocument();
    expect(screen.getAllByText("From GH issue?").length).toBeGreaterThan(0);
  });

  it("opens a mapped plan-running work order on the plan canvas", () => {
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup
            fixture={splitRunFixtureForWorkOrder({
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
            })}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: "Plan job" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-plan-0")).toBeInTheDocument();
    expect(screen.getByText("Planning")).toBeInTheDocument();
    expect(screen.getAllByText("From GH issue?").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("split-run-phase-implement")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("highlights the PR Closure log for the selected canvas component", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <WorkOrderSplitRunPopup
            fixture={{
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
            }}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const stream = screen.getByTestId("split-run-stream-done");
    expect(within(stream).queryByText("Started")).not.toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /merge-screenshot/ })).toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /#510/ })).toBeInTheDocument();

    await user.click(within(screen.getByTestId("split-run-stream-line-find-work-order")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-find-work-order")).toHaveAttribute("data-highlighted", "true");
  });
});
