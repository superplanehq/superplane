import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, mock } from "bun:test";
import type { SplitRunPhase } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";

const useSplitRunLiveCanvas = mock(() => ({
  enabled: false,
  isError: false,
  isLoading: false,
  canvas: undefined,
  stream: [],
  rootEventId: undefined,
}));
mock.module("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas }));
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
