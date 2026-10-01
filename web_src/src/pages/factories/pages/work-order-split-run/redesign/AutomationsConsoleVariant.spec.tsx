import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../../__fixtures__/lineMetricsBoardOrders";
import { OPEN_WORK_ORDER_CHECKS } from "../../../__fixtures__/workOrderCheckFixtures";
import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { LiveHeaderSpendProvider, useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import { buildSplitRunFooter } from "../splitRunFooter";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "../splitRunMocks";
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
    expect(screen.getByTestId("redesign-console-column-implement").getAttribute("data-completed")).toBe("true");
    expect(screen.getByTestId("redesign-console-column-verify").getAttribute("data-completed")).toBeNull();
  });

  it("lists merge confidence in Checks when the task has no verify step", () => {
    renderConsole(
      splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER, {
        checks: OPEN_WORK_ORDER_CHECKS,
        demoArtifacts: false,
      }),
    );

    const checks = within(screen.getByTestId("redesign-console-summary")).getByTestId("redesign-console-checks");
    expect(within(checks).getByText("Merge confidence")).toBeInTheDocument();
    expect(within(checks).queryByText("Confidence score")).not.toBeInTheDocument();
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

function cardState(name: string) {
  return screen.getByRole("button", { name: `Toggle ${name} details` }).closest("[data-state]");
}

describe("AutomationsConsoleVariant card collapse", () => {
  it("keeps a running card open and collapses finished cards", () => {
    renderConsole(SPLIT_RUN_RUNNING);

    expect(cardState("Implementation")).toHaveAttribute("data-state", "open");
    expect(cardState("Ingest")).toHaveAttribute("data-state", "closed");
  });

  it("collapses the last card after every automation finishes", () => {
    const { rerenderConsole } = renderConsole(SPLIT_RUN_RUNNING);
    expect(cardState("Implementation")).toHaveAttribute("data-state", "open");

    rerenderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "passed",
      footerTone: "done",
      phases: SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" })),
    });

    expect(cardState("Implementation")).toHaveAttribute("data-state", "closed");
    expect(cardState("Ingest")).toHaveAttribute("data-state", "closed");
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
