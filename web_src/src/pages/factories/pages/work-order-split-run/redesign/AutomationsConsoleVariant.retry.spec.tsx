import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within, act } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type * as ApiClient from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { showErrorToast } from "@/lib/toast";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { LiveHeaderSpendProvider } from "../liveHeaderSpendContext";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";

const { describeRunMock, reemitMock, listRunsMock } = vi.hoisted(() => ({
  describeRunMock: vi.fn(),
  reemitMock: vi.fn(),
  listRunsMock: vi.fn(),
}));

vi.mock("@/api-client", () => {
  const actual = unmockedSrc<typeof ApiClient>("api-client");
  return {
    ...actual,
    canvasesDescribeRun: (...args: unknown[]) => describeRunMock(...args),
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitMock(...args),
    canvasesListRuns: (...args: unknown[]) => listRunsMock(...args),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
  showInfoToast: vi.fn(),
}));

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
  return { ...view, client, rerenderConsole: (next: typeof fixture) => view.rerender(ui(next)) };
}

function previewPhase(status: "failed" | "passed" | "running") {
  return {
    id: "column-app-preview",
    name: "Verify",
    status,
    duration: "12s",
    startedAt: "2026-08-26T11:10:00Z",
    componentName: "Preview",
    description: "Preview deploy failed.",
    appId: "app-preview",
    runId: "run-preview",
    columnKey: "verify" as const,
    artifacts: [],
    stream: [],
    canvasSteps: [],
  };
}

function consoleWithPreview(status: "failed" | "passed" | "running") {
  return {
    ...SPLIT_RUN_RUNNING,
    phases: [...SPLIT_RUN_RUNNING.phases, previewPhase(status)],
  };
}

function openAutomation(name: string) {
  const toggle = screen.getByRole("button", { name: `Toggle ${name} details` });
  if (toggle.getAttribute("aria-expanded") !== "true") {
    fireEvent.click(toggle);
  }
}

describe("AutomationsConsoleVariant stage automation retry", () => {
  beforeEach(() => {
    describeRunMock.mockReset().mockResolvedValue({ data: {} });
    reemitMock.mockReset().mockResolvedValue({ data: {} });
    listRunsMock.mockReset().mockResolvedValue({ data: { runs: [] } });
    vi.mocked(showErrorToast).mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows Retry on a failed custom automation when the user can update the task", () => {
    renderConsole(consoleWithPreview("failed"), { canStopRun: true, organizationId: "org-1" });

    openAutomation("Preview");

    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
  });

  it("hides Retry when the user cannot update the task", () => {
    renderConsole(consoleWithPreview("failed"), { canStopRun: false, organizationId: "org-1" });

    openAutomation("Preview");

    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("hides Retry on a running or successful automation", () => {
    const { rerenderConsole } = renderConsole(SPLIT_RUN_RUNNING, { canStopRun: true });

    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();

    rerenderConsole(consoleWithPreview("passed"));
    openAutomation("Preview");
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();

    rerenderConsole(consoleWithPreview("running"));
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("restarts a failed line step through line dispatch", () => {
    const onRerunStep = vi.fn();
    renderConsole(
      {
        ...SPLIT_RUN_RUNNING,
        phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
          phase.id === "implement" ? { ...phase, status: "failed" as const } : phase,
        ),
      },
      { canStopRun: true, onRerunStep, organizationId: "org-1" },
    );

    openAutomation("Implementation");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(onRerunStep).toHaveBeenCalledWith(expect.objectContaining({ id: "implement", stepIndex: 0 }));
    expect(reemitMock).not.toHaveBeenCalled();
  });

  it("restarts a failed line step after its canvas run is removed", () => {
    const onRerunStep = vi.fn();
    renderConsole(
      {
        ...SPLIT_RUN_RUNNING,
        phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
          phase.id === "implement" ? { ...phase, status: "failed" as const, runId: undefined } : phase,
        ),
      },
      { canStopRun: true, onRerunStep, organizationId: "org-1" },
    );

    openAutomation("Implementation");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(onRerunStep).toHaveBeenCalledWith(expect.objectContaining({ id: "implement", stepIndex: 0 }));
    expect(reemitMock).not.toHaveBeenCalled();
  });

  it("re-emits the root trigger and keeps the failed run as an earlier attempt", async () => {
    describeRunMock.mockResolvedValue({
      data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
    });
    reemitMock.mockResolvedValue({ data: { eventId: "event-rerun" } });
    listRunsMock.mockResolvedValue({
      data: {
        runs: [{ id: "run-preview-2", rootEvent: { id: "event-rerun", nodeId: "trigger-preview" } }],
      },
    });
    const onRerunStep = vi.fn();
    const { client } = renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      onRerunStep,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });
    const invalidate = vi.spyOn(client, "invalidateQueries");

    const verify = screen.getByTestId("redesign-console-column-verify");
    expect(within(verify).getByText("1 agent run")).toBeInTheDocument();
    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => {
      expect(within(verify).getByText("2 agent runs")).toBeInTheDocument();
    });
    expect(describeRunMock).toHaveBeenCalledWith(
      expect.objectContaining({ path: { canvasId: "app-preview", runId: "run-preview" } }),
    );
    expect(reemitMock).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { canvasId: "app-preview", nodeId: "trigger-preview", eventId: "event-preview" },
      }),
    );
    expect(listRunsMock).toHaveBeenCalledWith(expect.objectContaining({ path: { canvasId: "app-preview" } }));
    expect(onRerunStep).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
    expect(within(verify).getAllByRole("button", { name: "Toggle Verify" })).toHaveLength(2);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["canvases"] });
  });

  it("shows an error and does not start a run when the root trigger is missing", async () => {
    describeRunMock.mockResolvedValue({ data: { run: { id: "run-preview" } } });
    renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });

    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("Failed to restart run");
    });
    expect(reemitMock).not.toHaveBeenCalled();
    expect(within(screen.getByTestId("redesign-console-column-verify")).getByText("1 agent run")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
  });

  it("disables Retry while the restart request is in progress", async () => {
    let releaseDescribe: (value: unknown) => void = () => undefined;
    describeRunMock.mockReturnValue(
      new Promise((resolve) => {
        releaseDescribe = resolve;
      }),
    );
    renderConsole(consoleWithPreview("failed"), { canStopRun: true, organizationId: "org-1" });

    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(screen.getByRole("button", { name: "Retry" })).toBeDisabled();
    releaseDescribe({
      data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
    });
    await waitFor(() => {
      expect(reemitMock).toHaveBeenCalled();
    });
  });

  it("shows Retry again when the restarted run fails before the task updates", async () => {
    describeRunMock.mockResolvedValue({
      data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
    });
    reemitMock.mockResolvedValue({ data: { eventId: "event-rerun" } });
    listRunsMock.mockResolvedValue({
      data: {
        runs: [
          {
            id: "run-preview-2",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            rootEvent: { id: "event-rerun", nodeId: "trigger-preview" },
          },
        ],
      },
    });
    renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });

    const verify = screen.getByTestId("redesign-console-column-verify");
    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => {
      expect(within(verify).getByText("2 agent runs")).toBeInTheDocument();
    });
    openAutomation("Preview");
    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();

    describeRunMock.mockResolvedValue({
      data: { run: { id: "run-preview-2", rootEvent: { id: "event-rerun-2", nodeId: "trigger-preview" } } },
    });
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => {
      expect(describeRunMock).toHaveBeenCalledWith(
        expect.objectContaining({ path: { canvasId: "app-preview", runId: "run-preview-2" } }),
      );
    });
  });

  it("keeps a second retry visible after the first local attempt is stored", async () => {
    describeRunMock.mockResolvedValue({
      data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
    });
    reemitMock.mockResolvedValueOnce({ data: { eventId: "event-rerun" } });
    listRunsMock.mockResolvedValue({
      data: {
        runs: [
          {
            id: "run-preview-2",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            rootEvent: { id: "event-rerun", nodeId: "trigger-preview" },
          },
        ],
      },
    });
    const { rerenderConsole } = renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });

    const verify = screen.getByTestId("redesign-console-column-verify");
    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => {
      expect(within(verify).getByText("2 agent runs")).toBeInTheDocument();
    });
    await waitFor(() => {
      openAutomation("Preview");
      expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
    });

    describeRunMock.mockImplementation(async (args: { path?: { runId?: string } }) => {
      if (args.path?.runId === "run-preview-2") {
        return {
          data: { run: { id: "run-preview-2", rootEvent: { id: "event-rerun", nodeId: "trigger-preview" } } },
        };
      }
      return {
        data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
      };
    });
    reemitMock.mockResolvedValueOnce({ data: { eventId: "event-rerun-2" } });
    listRunsMock.mockResolvedValue({ data: { runs: [] } });
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => {
      expect(within(verify).getByText("3 agent runs")).toBeInTheDocument();
    });

    rerenderConsole({
      ...consoleWithPreview("failed"),
      phases: [
        ...consoleWithPreview("failed").phases,
        {
          ...previewPhase("failed"),
          id: "stored-preview-2",
          runId: "run-preview-2",
          startedAt: "2026-08-26T11:12:00Z",
        },
      ],
    });

    expect(within(verify).getByText("3 agent runs")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("keeps Retry closed and keeps looking when the new run is late", async () => {
    vi.useFakeTimers();
    describeRunMock.mockResolvedValue({
      data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
    });
    reemitMock.mockResolvedValue({ data: { eventId: "event-rerun" } });
    listRunsMock
      .mockRejectedValueOnce(new Error("unavailable"))
      .mockResolvedValueOnce({ data: { runs: [] } })
      .mockResolvedValue({
        data: {
          runs: [
            {
              id: "run-preview-2",
              state: "STATE_STARTED",
              rootEvent: { id: "event-rerun", nodeId: "trigger-preview" },
            },
          ],
        },
      });
    renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });

    const verify = screen.getByTestId("redesign-console-column-verify");
    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(within(verify).getByText("2 agent runs")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
    expect(reemitMock).toHaveBeenCalledTimes(1);
    expect(listRunsMock).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(listRunsMock).toHaveBeenCalledTimes(2);
    expect(reemitMock).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(listRunsMock.mock.calls.length).toBeGreaterThanOrEqual(3);
    expect(reemitMock).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("follows a found retry by run id after newer runs fill the list", async () => {
    vi.useFakeTimers();
    describeRunMock.mockImplementation(async (args: { path?: { runId?: string } }) => {
      if (args.path?.runId === "run-preview-2") {
        return {
          data: {
            run: {
              id: "run-preview-2",
              state: "STATE_FINISHED",
              result: "RESULT_FAILED",
            },
          },
        };
      }
      return {
        data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
      };
    });
    reemitMock.mockResolvedValue({ data: { eventId: "event-rerun" } });
    listRunsMock.mockResolvedValueOnce({
      data: {
        runs: [
          {
            id: "run-preview-2",
            state: "STATE_STARTED",
            rootEvent: { id: "event-rerun", nodeId: "trigger-preview" },
          },
        ],
      },
    });
    listRunsMock.mockResolvedValue({ data: { runs: [] } });
    renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });

    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(listRunsMock).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(describeRunMock).toHaveBeenCalledWith(
      expect.objectContaining({ path: { canvasId: "app-preview", runId: "run-preview-2" } }),
    );
    expect(listRunsMock).toHaveBeenCalledTimes(1);
    openAutomation("Preview");
    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
  });

  it("does not poll for a retry after the task closes", async () => {
    vi.useFakeTimers();
    let releaseDescribe: (value: unknown) => void = () => undefined;
    describeRunMock.mockReturnValue(
      new Promise((resolve) => {
        releaseDescribe = resolve;
      }),
    );
    reemitMock.mockResolvedValue({ data: { eventId: "event-rerun" } });
    const { unmount } = renderConsole(consoleWithPreview("failed"), {
      canStopRun: true,
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
    });

    openAutomation("Preview");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    unmount();
    await act(async () => {
      releaseDescribe({
        data: { run: { id: "run-preview", rootEvent: { id: "event-preview", nodeId: "trigger-preview" } } },
      });
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(reemitMock).toHaveBeenCalledTimes(1);
    expect(listRunsMock).not.toHaveBeenCalled();
  });
});
