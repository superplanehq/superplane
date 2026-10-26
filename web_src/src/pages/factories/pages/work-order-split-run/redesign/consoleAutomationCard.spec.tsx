import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, mock, vi } from "bun:test";
import { splitRunFixtureForWorkOrder, type SplitRunPhase } from "../splitRunMocks";
import { agentRunnerQueue } from "../agentRunnerQueue";
import { stageFromPhase } from "./automationsViewModel";
import type { AutomationStage } from "./automationsViewModel";
import * as canvasData from "@/hooks/useCanvasData";

const useSplitRunLiveCanvas = mock(() => ({
  enabled: false,
  isError: false,
  isLoading: false,
  canvas: undefined,
  stream: [],
  rootEventId: undefined,
}));
const useEventExecutions = mock(() => ({
  data: undefined as { executions?: unknown } | undefined,
  isError: false,
  isLoading: false,
}));
mock.module("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas }));
mock.module("@/hooks/useCanvasData", () => ({
  ...canvasData,
  useEventExecutions,
}));
function captureRun(id: string, status: AutomationStage["status"] = "running"): AutomationStage {
  return {
    id,
    name: id,
    appId: "canvas-implement",
    componentName: "Implement",
    status,
    statusLabel: "Running",
    duration: "40m",
    checks: [],
    outputs: { pullRequests: [], artifacts: [] },
    plumbing: [],
    agentSteps: [],
    steps: [],
    rawLog: "",
  };
}
function checksPhase(): SplitRunPhase {
  return {
    id: "implement",
    name: "Implement",
    status: "running",
    duration: "",
    componentName: "Implement",
    artifacts: [],
    stream: [],
    canvasSteps: [],
    appId: "canvas-implement",
    runId: "run-implement",
  };
}

const { ConsoleAutomationCard } = await import("./consoleAutomationCard");

afterEach(() => {
  vi.useRealTimers();
});

describe("automation run time", () => {
  it("keeps the same duration basis while live, after finishing, and when reopened", async () => {
    const now = Date.now();
    vi.useFakeTimers();
    vi.setSystemTime(now);
    let run = { ...captureRun("analysis"), duration: "28m 20s" };
    let phase: SplitRunPhase = {
      ...checksPhase(),
      id: run.id,
      name: "Analysis",
      status: run.status,
      duration: run.duration,
      durationRunning: true,
      // Analysis can combine attempts separated by a long wait.
      startedAt: new Date(now - 6 * 60 * 60 * 1000).toISOString(),
    };
    useSplitRunLiveCanvas.mockImplementation(() => ({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [],
      rootEventId: undefined,
      agentQueue: agentRunnerQueue(
        { spec: { nodes: [{ id: "agent", component: "runnerClaudeCode", type: "TYPE_ACTION" }] } },
        {
          state: run.status === "passed" ? "STATE_FINISHED" : "STATE_STARTED",
          executions: [
            {
              nodeId: "agent",
              state: run.status === "passed" ? "STATE_FINISHED" : "STATE_STARTED",
              createdAt: new Date(now - 30 * 1000).toISOString(),
              updatedAt: run.status === "passed" ? new Date(now + 3000).toISOString() : undefined,
            },
          ],
        },
      ),
    }));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const ui = () => (
      <QueryClientProvider client={client}>
        <ConsoleAutomationCard
          automation={{ id: "analysis", name: "Backlog", latest: run, runs: [run] }}
          phase={phase}
          phases={[phase]}
          organizationId="org-1"
          expandIdle={false}
          canStopRun={false}
          actionBusy={false}
        />
      </QueryClientProvider>
    );
    const view = render(ui());
    const header = screen.getByTestId("redesign-console-card-header-analysis");
    expect(header).toHaveTextContent("28m 20s");
    expect(header).toHaveTextContent("Run time");

    await act(async () => {
      vi.advanceTimersByTime(2000);
    });
    expect(header).toHaveTextContent("28m 22s");

    view.rerender(ui());
    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(header).toHaveTextContent("28m 23s");

    run = { ...run, duration: "28m 30s" };
    phase = { ...phase, duration: run.duration };
    view.rerender(ui());
    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(header).toHaveTextContent("28m 31s");

    run = { ...run, status: "passed", duration: "28m 35s" };
    phase = { ...phase, status: run.status, duration: run.duration, durationRunning: false };
    view.rerender(ui());
    await act(async () => {
      vi.advanceTimersByTime(60000);
    });
    expect(header).toHaveTextContent("28m 35s");

    view.unmount();
    render(ui());
    expect(screen.getByTestId("redesign-console-card-header-analysis")).toHaveTextContent("28m 35s");
  });

  it.each(["running", "waiting", "cancelled"] as const)(
    "keeps a %s phase fixed when its duration has stopped",
    async (status) => {
      const now = Date.now();
      vi.useFakeTimers();
      vi.setSystemTime(now);
      const run = { ...captureRun("analysis", status), duration: "5m" };
      const phase = { ...checksPhase(), id: run.id, status, duration: run.duration, durationRunning: false };
      useSplitRunLiveCanvas.mockImplementation(() => ({
        enabled: true,
        isError: false,
        isLoading: false,
        canvas: undefined,
        stream: [],
        rootEventId: undefined,
        agentQueue: agentRunnerQueue(
          { spec: { nodes: [{ id: "agent", component: "runnerClaudeCode", type: "TYPE_ACTION" }] } },
          {
            state: "STATE_STARTED",
            executions: [{ nodeId: "agent", state: "STATE_STARTED", createdAt: new Date(now - 30000).toISOString() }],
          },
        ),
      }));
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(
        <QueryClientProvider client={client}>
          <ConsoleAutomationCard
            automation={{ id: "analysis", name: "Backlog", latest: run, runs: [run] }}
            phase={phase}
            phases={[phase]}
            organizationId="org-1"
            expandIdle={false}
            canStopRun={false}
            actionBusy={false}
          />
        </QueryClientProvider>,
      );
      const header = screen.getByTestId("redesign-console-card-header-analysis");
      expect(header).toHaveTextContent("5m");
      const fixedHeader = header.textContent;
      await act(async () => {
        vi.advanceTimersByTime(60000);
      });
      expect(header.textContent).toBe(fixedHeader);
    },
  );
});

describe("stopped automation card", () => {
  it("keeps the cancelled run neutral in the header and history row after the stop request finishes", () => {
    const fixture = splitRunFixtureForWorkOrder(
      { id: "task", state: "STATE_OPEN" },
      {
        demoArtifacts: false,
        prFeedbackRuns: [
          {
            canvasId: "visual-evidence",
            handlerName: "Visual Evidence",
            title: "Capture screenshots or a short video",
            run: { id: "stopped", state: "STATE_FINISHED", result: "RESULT_CANCELLED" },
          },
        ],
      },
    );
    const phase = fixture.phases.find((entry) => entry.runId === "stopped")!;
    const latest = stageFromPhase(phase);
    const previous = captureRun("previous", "passed");
    const view = render(
      <ConsoleAutomationCard
        automation={{ id: "evidence", name: "Visual Evidence", latest, runs: [previous, latest] }}
        phase={phase}
        phases={[phase]}
        expandIdle={false}
        canStopRun
        actionBusy={false}
      />,
    );

    const header = screen.getByTestId("redesign-console-card-header-evidence");
    expect(header).toHaveTextContent("Stopped");
    fireEvent.click(screen.getByRole("button", { name: "Toggle Visual Evidence details" }));
    expect(screen.getAllByText("Stopped")).toHaveLength(2);
    expect(view.container.querySelectorAll(".bg-\\[color\\:var\\(--status-cancelled-dot\\)\\]")).toHaveLength(2);
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
  });
});

describe("queued automation card", () => {
  it("shows one queue label and Cancel, then restores the run details when work starts", async () => {
    let queued = true;
    useSplitRunLiveCanvas.mockImplementation(() => ({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [],
      rootEventId: undefined,
      agentQueue: { queued },
    }));
    const run = captureRun("queued");
    const phase = { ...checksPhase(), id: run.id, status: run.status };
    const onStopRun = mock(() => {});
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const ui = () => (
      <QueryClientProvider client={client}>
        <ConsoleAutomationCard
          automation={{ id: "implement", name: "Implement", latest: run, runs: [run] }}
          phase={phase}
          phases={[phase]}
          organizationId="org-1"
          expandIdle={false}
          canStopRun
          actionBusy={false}
          onStopRun={onStopRun}
        />
      </QueryClientProvider>
    );
    const view = render(ui());
    await waitFor(() => expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument());
    expect(screen.getAllByText("Queued")).toHaveLength(1);
    expect(screen.getByText("Waiting for an available agent slot.")).toBeInTheDocument();
    expect(screen.queryByText("40m")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();
    queued = false;
    view.rerender(ui());
    await waitFor(() => expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument());
    expect(screen.queryByText("Waiting for an available agent slot.")).not.toBeInTheDocument();
    expect(screen.queryByText("Queued")).not.toBeInTheDocument();
    queued = true;
    view.rerender(ui());
    await waitFor(() => expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onStopRun).toHaveBeenCalledWith({ appId: phase.appId, runId: phase.runId });
  });

  it("keeps a repair description and check page beside the queue message", async () => {
    useSplitRunLiveCanvas.mockImplementation(() => ({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: {
        nodes: [{ id: "wait-checks", component: "github.waitForPullRequestChecks" }],
      },
      stream: [],
      rootEventId: "event-root",
      agentQueue: { queued: true },
    }));
    useEventExecutions.mockReturnValue({
      data: {
        executions: [
          {
            nodeId: "wait-checks",
            state: "STATE_FINISHED",
            metadata: {
              selectedChecks: [
                {
                  name: "build",
                  description: "CI",
                  status: "completed",
                  conclusion: "failure",
                  detailsUrl: "https://example.com/build",
                  summary: "The build failed on Semaphore 2.0.",
                },
              ],
            },
          },
        ],
      },
      isError: false,
      isLoading: false,
    });
    const run = {
      ...captureRun("repair"),
      name: "Fixing failed checks on abc",
      description: "Repair the failed build.",
      checks: [
        {
          id: "build",
          name: "Build",
          score: 0,
          maxScore: 1,
          format: "boolean" as const,
          level: "critical" as const,
          summary: "The build failed.",
        },
      ],
    };
    const phase = { ...checksPhase(), id: run.id, status: run.status };
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <ConsoleAutomationCard
          automation={{ id: "checks", name: "Fix pull request checks", latest: run, runs: [run] }}
          phase={phase}
          phases={[phase]}
          organizationId="org-1"
          expandIdle={false}
          canStopRun
          actionBusy={false}
          onStopRun={() => {}}
        />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument());
    expect(screen.getByText("Waiting for an available agent slot.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "build: CI" })).toHaveAttribute("href", "https://example.com/build");
    expect(screen.getByText("Repair the failed build.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: /Checks/ }));
    expect(screen.getByText("Build")).toBeInTheDocument();
    expect(screen.getByText("The build failed.")).toBeInTheDocument();
  });
});

describe("queued automation history", () => {
  it("keeps the previous run accessible beside the queued run", async () => {
    useSplitRunLiveCanvas.mockImplementation((_organizationId: unknown, phase?: SplitRunPhase) => ({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [],
      rootEventId: undefined,
      agentQueue: { queued: phase?.id === "queued" },
    }));
    const previous = { ...captureRun("previous", "passed"), description: "Previous implementation output." };
    const latest = captureRun("queued");
    const phases = [previous, latest].map((run) => ({ ...checksPhase(), id: run.id, status: run.status }));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <ConsoleAutomationCard
          automation={{ id: "implement", name: "Implement", latest, runs: [previous, latest] }}
          phase={phases[1]}
          phases={phases}
          organizationId="org-1"
          expandIdle={false}
          canStopRun
          actionBusy={false}
          onStopRun={() => {}}
        />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Toggle queued" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Toggle previous" }));
    expect(screen.getByText("Previous implementation output.")).toBeInTheDocument();
    expect(screen.getByText("Waiting for an available agent slot.")).toBeInTheDocument();
  });
});
