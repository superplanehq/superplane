import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../../__fixtures__/lineMetricsBoardOrders";
import { OPEN_WORK_ORDER_CHECKS } from "../../../__fixtures__/workOrderCheckFixtures";
import { TWO_HOURS_AGO } from "../../../__fixtures__/factoryPageIds";
import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { LiveHeaderSpendProvider, useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import { buildSplitRunFooter } from "../splitRunFooter";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "../splitRunMocks";
import { CREATED_MANUALLY } from "../splitRunSource";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";

type ConsoleProps = Parameters<typeof AutomationsConsoleVariant>[0];

function renderConsole(fixture: typeof SPLIT_RUN_RUNNING, extra: Partial<ConsoleProps> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui = (next: typeof fixture) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LiveHeaderSpendProvider>
              <AutomationsConsoleVariant fixture={next} source={next.source} {...extra} />
            </LiveHeaderSpendProvider>
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(ui(fixture));
  return { ...view, rerenderConsole: (next: typeof fixture) => view.rerender(ui(next)) };
}

describe("AutomationsConsoleVariant timeline markers", () => {
  it("fills finished columns and spins the live column", () => {
    renderConsole(SPLIT_RUN_RUNNING);

    expect(screen.getByTestId("redesign-console-column-marker-backlog")).toHaveAttribute("data-status", "completed");
    expect(screen.getByTestId("redesign-console-column-marker-backlog")).toHaveTextContent("Completed");
    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveAttribute("data-status", "running");
    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveTextContent("Running");
    expect(screen.getByTestId("redesign-console-column-marker-verify")).toHaveAttribute("data-status", "pending");
    expect(screen.getByTestId("redesign-console-column-marker-done")).toHaveAttribute("data-status", "pending");
    expect(
      within(screen.getByTestId("redesign-console-column-backlog")).queryByText("Skipped"),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-column-implement").getAttribute("data-completed")).toBe("true");
    expect(screen.getByTestId("redesign-console-column-verify").getAttribute("data-completed")).toBeNull();
  });

  it("lists Risk score in Checks when the task has no verify step", () => {
    renderConsole(
      splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER, {
        checks: OPEN_WORK_ORDER_CHECKS,
        demoArtifacts: false,
      }),
    );

    const checks = within(screen.getByTestId("redesign-console-summary")).getByTestId("redesign-console-checks");
    const header = within(checks).getByRole("button", { name: "Merge confidence" });
    expect(header).toHaveAttribute("aria-expanded", "false");
    expect(within(checks).queryByText("Blast radius")).not.toBeInTheDocument();
    fireEvent.click(header);
    expect(within(checks).getByText("Blast radius")).toBeInTheDocument();
    expect(within(checks).queryByText("Confidence score")).not.toBeInTheDocument();
  });

  it("keeps planning review scores off Merge confidence", () => {
    renderConsole(
      splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
        demoArtifacts: false,
        checks: [
          {
            id: "check-confidence",
            key: "confidence",
            name: "Confidence score",
            score: 2,
            maxScore: 5,
            level: "LEVEL_CAUTION",
          },
          {
            id: "check-clarity",
            key: "clarity",
            name: "Clarity score",
            score: 3,
            maxScore: 3,
            level: "LEVEL_POSITIVE",
          },
          {
            id: "check-complexity",
            key: "complexity",
            name: "Complexity",
            score: 2,
            maxScore: 3,
            level: "LEVEL_CAUTION",
          },
          {
            id: "check-verifiability",
            key: "verifiability",
            name: "Verifiability",
            score: 3,
            maxScore: 3,
            level: "LEVEL_POSITIVE",
          },
        ],
        analysisRuns: [
          {
            canvasId: "canvas-backlog",
            workOrderId: DRAFT_WORK_ORDER.id ?? "",
            run: {
              id: "run-backlog",
              canvasId: "canvas-backlog",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-28T12:00:00Z",
              finishedAt: "2026-08-28T12:00:20Z",
            },
          },
        ],
      }),
    );

    const summary = screen.getByTestId("redesign-console-summary");
    expect(within(summary).queryByTestId("redesign-console-checks")).not.toBeInTheDocument();
    expect(within(summary).queryByRole("button", { name: "Merge confidence" })).not.toBeInTheDocument();
    expect(within(summary).queryByText("Clarity")).not.toBeInTheDocument();
    expect(within(summary).queryByText("Complexity")).not.toBeInTheDocument();
    expect(within(summary).queryByText("Verifiability")).not.toBeInTheDocument();
  });

  it("lists each column app that ran for the task as its own card", () => {
    renderConsole(
      splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        demoArtifacts: false,
        checks: [
          {
            id: "check-risk",
            key: "risk-review",
            name: "Risk score",
            score: 2,
            maxScore: 5,
            level: "LEVEL_POSITIVE",
            automation: { appId: "app-risk", appName: "Risk score" },
            runId: "run-risk",
            updatedAt: "2026-08-26T11:10:00Z",
          },
        ],
        columnApps: [
          { id: "app-risk", name: "Risk score", columnKey: "verify" },
          { id: "app-storybook", name: "Deploys Storybook", columnKey: "verify" },
        ],
        prFeedbackRuns: [
          {
            canvasId: "app-storybook",
            title: "Storybook deployment ready",
            pullRequestNumber: "12",
            run: {
              id: "run-storybook",
              canvasId: "app-storybook",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T11:30:00Z",
            },
          },
          {
            canvasId: "canvas-comment",
            handlerName: "Address PR feedback",
            title: "Address new review comment",
            pullRequestNumber: "12",
            run: {
              id: "run-comment",
              canvasId: "canvas-comment",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T12:00:00Z",
            },
          },
          {
            canvasId: "canvas-checks",
            handlerName: "Fix pull request checks",
            title: "Fix failing checks",
            pullRequestNumber: "12",
            run: {
              id: "run-checks",
              canvasId: "canvas-checks",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T10:00:00Z",
            },
          },
        ],
      }),
    );

    const verify = screen.getByTestId("redesign-console-column-verify");
    expect(within(verify).getByText("Fix pull request checks")).toBeInTheDocument();
    expect(within(verify).getByText("Address PR feedback")).toBeInTheDocument();
    expect(within(verify).getByText("Deploys Storybook")).toBeInTheDocument();
    expect(within(verify).getByText("Risk score")).toBeInTheDocument();
  });

  it("marks a failed implement column", () => {
    renderConsole(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER));

    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveAttribute("data-status", "failed");
    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveTextContent("Failed");
    expect(screen.getByTestId("redesign-console-column-implement").getAttribute("data-completed")).toBe("true");
  });

  it("marks a stopped implement column as canceled", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "cancelled",
      footerTone: "stopped",
      footer: buildSplitRunFooter({ kind: "stopped" }),
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement" ? { ...phase, status: "cancelled" as const } : phase,
      ),
    });

    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveAttribute("data-status", "cancelled");
    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveTextContent("Canceled");
    expect(screen.getByTestId("redesign-console-column-implement").getAttribute("data-completed")).toBe("true");
  });

  it("marks a waiting implement column as waiting", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "waiting",
      footerTone: "waiting",
      footer: buildSplitRunFooter({ kind: "waiting" }),
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement" ? { ...phase, status: "waiting" as const } : phase,
      ),
    });

    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveAttribute("data-status", "waiting");
    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveTextContent("Waiting");
  });

  it("keeps a pending implement column as not started", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "pending",
      footerTone: "draft",
      footer: buildSplitRunFooter({ kind: "draft" }),
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement" ? { ...phase, status: "pending" as const } : phase,
      ),
    });

    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveAttribute("data-status", "pending");
    expect(screen.getByTestId("redesign-console-column-marker-implement")).toHaveTextContent("Not started");
    expect(screen.getByTestId("redesign-console-column-implement").getAttribute("data-completed")).toBeNull();
  });
});

describe("AutomationsConsoleVariant summary strip", () => {
  it("shows the live note with Stop while a run is active", () => {
    const onStopRun = vi.fn();
    renderConsole(SPLIT_RUN_RUNNING, { canStopRun: true, onStopRun });

    const note = screen.getByTestId("redesign-console-live-note");
    expect(note).toHaveTextContent("Implement is running");
    fireEvent.click(screen.getByTestId("redesign-console-stop-run"));
    expect(onStopRun).toHaveBeenCalledWith(SPLIT_RUN_RUNNING.footer.run);
  });

  it("hides Stop when this person cannot stop runs", () => {
    renderConsole(SPLIT_RUN_RUNNING, { onStopRun: vi.fn() });

    expect(screen.getByTestId("redesign-console-live-note")).toBeInTheDocument();
    expect(screen.queryByTestId("redesign-console-stop-run")).not.toBeInTheDocument();
  });

  it("shows the live note instead of an empty review band while running", () => {
    renderConsole(SPLIT_RUN_RUNNING, { panelReview: <div data-testid="review-stub" /> });

    expect(screen.queryByTestId("review-stub")).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-live-note")).toBeInTheDocument();
  });

  it("shows the waiting note when the task waits with no note and no live run", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "waiting",
      footerTone: "waiting",
      footer: buildSplitRunFooter({ kind: "waiting" }),
      phases: SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" })),
    });

    const note = screen.getByTestId("redesign-console-waiting-note");
    expect(note).toHaveTextContent("This task is waiting");
    expect(screen.queryByTestId("redesign-console-live-note")).not.toBeInTheDocument();
  });

  it("lists task artifacts of every type, oldest first", () => {
    renderConsole(SPLIT_RUN_RUNNING, {
      artifacts: [
        {
          id: "art-link",
          type: "TYPE_LINK",
          data: { name: "Design doc", url: "https://example.com/doc" },
          createdAt: new Date(Date.now() - 60_000).toISOString(),
        },
        {
          id: "art-md",
          type: "TYPE_MARKDOWN",
          data: { name: "report.md", body: "# Findings" },
          createdAt: new Date(Date.now() - 180_000).toISOString(),
        },
        {
          id: "art-spec",
          type: "TYPE_MARKDOWN",
          data: { name: "spec.md", body: "# Spec" },
          createdAt: new Date(Date.now() - 240_000).toISOString(),
        },
        {
          id: "art-file",
          type: "TYPE_FILE",
          data: { filename: "trace.log", url: "https://example.com/trace.log" },
          createdAt: new Date(Date.now() - 120_000).toISOString(),
        },
      ],
    });

    const section = screen.getByTestId("redesign-console-artifacts");
    const text = section.textContent ?? "";
    expect(text).toContain("spec.md");
    expect(text).toContain("report.md");
    expect(text).toContain("trace.log");
    expect(text).toContain("Design doc");
    expect(text.indexOf("spec.md")).toBeLessThan(text.indexOf("report.md"));
    expect(text.indexOf("report.md")).toBeLessThan(text.indexOf("trace.log"));
    expect(text.indexOf("trace.log")).toBeLessThan(text.indexOf("Design doc"));
  });

  it("shows the decision note when the footer carries one", () => {
    renderConsole(
      {
        ...SPLIT_RUN_RUNNING,
        lineStatus: "passed",
        footerTone: "done",
        footer: buildSplitRunFooter({ kind: "done", status: "completed" }),
        phases: SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" })),
      },
      { panelReview: <div data-testid="review-stub" /> },
    );

    expect(screen.getByTestId("review-stub")).toBeInTheDocument();
    expect(screen.queryByTestId("redesign-console-waiting-note")).not.toBeInTheDocument();
  });
});

describe("AutomationsConsoleVariant intake", () => {
  it("shows Intake with the source icon and the task description", () => {
    renderConsole(SPLIT_RUN_RUNNING, { taskDescription: "Users see duplicate refund entries." });

    const intake = screen.getByTestId("redesign-console-column-intake");
    expect(within(intake).getByText("Intake")).toBeInTheDocument();
    expect(within(intake).getByTestId("redesign-console-column-marker-intake")).toHaveTextContent("GitHub");
    const description = within(intake).getByTestId("redesign-console-task-description");
    expect(description).toHaveTextContent("Users see duplicate refund entries.");
    expect(within(description).getByTestId("split-run-source-ticket")).toHaveTextContent("acme/payments-service#103");
    expect(within(description).getByTestId("redesign-console-intake-added-by")).toHaveTextContent(
      "Intake GitHub issues",
    );
    expect(screen.queryByRole("button", { name: "Toggle Ingest details" })).not.toBeInTheDocument();
    expect(
      intake.compareDocumentPosition(screen.getByTestId("redesign-console-column-backlog")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeGreaterThan(0);
  });

  it("keeps Intake and hides the empty sentence when edit is not allowed", () => {
    renderConsole(SPLIT_RUN_RUNNING, { taskDescription: "   ", canEditDescription: false });

    const description = screen.getByTestId("redesign-console-task-description");
    expect(within(description).getByTestId("split-run-source-ticket")).toHaveTextContent("acme/payments-service#103");
    expect(screen.queryByText("No description yet.")).not.toBeInTheDocument();
  });

  it("hides Intake when there is no source and edit is not allowed", () => {
    renderConsole(SPLIT_RUN_RUNNING, { taskDescription: "   ", canEditDescription: false, source: undefined });

    expect(screen.queryByTestId("redesign-console-column-intake")).not.toBeInTheDocument();
    expect(screen.queryByTestId("redesign-console-task-description")).not.toBeInTheDocument();
  });

  it("keeps the empty state and edit control when edit is allowed", () => {
    renderConsole(SPLIT_RUN_RUNNING, { taskDescription: "", canEditDescription: true });

    const description = screen.getByTestId("redesign-console-task-description");
    expect(description).toHaveTextContent("No description yet.");
    expect(within(description).getByTestId("split-run-description-edit")).toBeInTheDocument();
    expect(within(description).getByTestId("split-run-source-ticket")).toHaveTextContent("acme/payments-service#103");
    expect(
      screen
        .getByTestId("redesign-console-column-intake")
        .compareDocumentPosition(screen.getByTestId("redesign-console-column-backlog")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeGreaterThan(0);
  });

  it("shows who added a manual task in the Intake description", () => {
    renderConsole(SPLIT_RUN_RUNNING, {
      source: { kind: "manual", person: SPLIT_RUN_RUNNING.owner, detail: CREATED_MANUALLY },
      taskDescription: "Users see duplicate refund entries.",
    });

    const description = screen.getByTestId("redesign-console-task-description");
    expect(within(description).getByTestId("split-run-source")).toHaveTextContent(SPLIT_RUN_RUNNING.owner.name);
    expect(within(description).getByTestId("redesign-console-intake-added-by")).toHaveTextContent(CREATED_MANUALLY);
    expect(within(description).queryByTestId("split-run-source-ticket")).not.toBeInTheDocument();
  });

  it("shows who imported a task in the Intake description", () => {
    const imported = splitRunFixtureForWorkOrder({
      ...OPEN_WORK_ORDER,
      createdBy: { user: { id: SPLIT_RUN_RUNNING.owner.id, name: SPLIT_RUN_RUNNING.owner.name } },
      origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
    });
    renderConsole(imported, { taskDescription: "Users see duplicate refund entries." });

    const description = screen.getByTestId("redesign-console-task-description");
    expect(within(description).getByTestId("split-run-source-ticket")).toHaveTextContent("acme/payments#12");
    expect(within(description).getByTestId("redesign-console-intake-added-by")).toHaveTextContent(
      `Imported by ${SPLIT_RUN_RUNNING.owner.name}`,
    );
  });
});

describe("AutomationsConsoleVariant column timing", () => {
  const arrived = formatWorkOrderDateTime(new Date(TWO_HOURS_AGO));

  it("stamps Intake without dwell and shows backlog arrival plus time spent", () => {
    renderConsole(SPLIT_RUN_RUNNING, { taskDescription: "Users see duplicate refund entries." });

    const intake = screen.getByTestId("redesign-console-column-timing-intake");
    expect(intake).toHaveTextContent(`Arrived ${arrived}`);
    expect(intake).not.toHaveTextContent("Spent");

    const backlog = screen.getByTestId("redesign-console-column-timing-backlog");
    expect(backlog).toHaveTextContent(`Arrived ${arrived}`);
    expect(backlog).toHaveTextContent("Spent 1h");

    expect(screen.queryByTestId("redesign-console-column-timing-verify")).not.toBeInTheDocument();
    expect(screen.queryByTestId("redesign-console-column-timing-done")).not.toBeInTheDocument();
  });

  it("shows Verify arrival and time so far while the task waits in that column", () => {
    const entered = "2026-09-30T12:00:00.000Z";
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "waiting",
      currentStepIndex: 0,
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement" ? { ...phase, status: "passed" as const, endedAt: entered } : phase,
      ),
    });

    const verify = screen.getByTestId("redesign-console-column-timing-verify");
    expect(verify).toHaveTextContent(`Arrived ${formatWorkOrderDateTime(new Date(entered))}`);
    expect(verify).toHaveTextContent("Spent");
  });
});

describe("AutomationsConsoleVariant run footer", () => {
  it("shows recorded spend and model on the open card footer", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement"
          ? { ...phase, costCents: "45", totalTokens: "2700", startedAt: "2026-09-30T00:59:00.000Z" }
          : phase,
      ),
    });

    const footer = screen.getByTestId("redesign-console-run-footer-implement");
    expect(footer).toHaveTextContent(formatWorkOrderDateTime(new Date("2026-09-30T00:59:00.000Z")));
    expect(footer).not.toHaveTextContent("Started");
    expect(footer).toHaveTextContent("$0.45");
    expect(footer).toHaveTextContent("2.7k");
    expect(footer).toHaveTextContent("sonnet 4-6");
  });

  it("updates the footer spend from live telemetry", () => {
    function ReportLiveSpend() {
      useReportLiveHeaderSpend("implement:agent", 2100, 45);
      return (
        <AutomationsConsoleVariant
          fixture={SPLIT_RUN_RUNNING}
          source={SPLIT_RUN_RUNNING.source}
          organizationId="org-1"
        />
      );
    }

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <LiveHeaderSpendProvider>
                <ReportLiveSpend />
              </LiveHeaderSpendProvider>
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const footer = screen.getByTestId("redesign-console-run-footer-implement");
    expect(footer).toHaveTextContent("$0.45");
    expect(footer).toHaveTextContent("2.1k");
    expect(footer).toHaveTextContent("sonnet 4-6");
  });
});
