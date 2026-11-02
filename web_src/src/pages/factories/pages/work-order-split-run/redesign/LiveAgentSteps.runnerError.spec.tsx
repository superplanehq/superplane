import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, expect, it, vi } from "vitest";

import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { useCanvas, useDescribeRun, useEventExecutions } from "@/hooks/useCanvasData";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";

import { AgentActivityView } from "../AgentActivityView";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { idleStream, implementPhase, implementStage } from "./LiveAgentSteps.testHelpers";

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: vi.fn(),
  useDescribeRun: vi.fn(),
  useEventExecutions: vi.fn(),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: vi.fn() }));
vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({ useLiveLogStream: vi.fn() }));

const FAILURE_MESSAGE = "Your ChatGPT session has expired. Sign in again to continue.";
const RUN_HREF = "/org/workspaces/example/automations/app/runs/run-1";

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(useCanvas).mockReturnValue({
    data: {
      spec: {
        nodes: [{ id: "agent", name: "Run Codex", component: "runnerCodex", type: "TYPE_ACTION" }],
      },
    },
    isLoading: false,
    isError: false,
  } as unknown as ReturnType<typeof useCanvas>);
  vi.mocked(useDescribeRun).mockReturnValue({
    data: {
      run: {
        id: "run-1",
        result: "RESULT_FAILED",
        rootEvent: { id: "event-1" },
        executions: [{ id: "execution-1", nodeId: "agent", state: "STATE_FINISHED", result: "RESULT_PASSED" }],
      },
    },
    isLoading: false,
    isError: false,
  } as unknown as ReturnType<typeof useDescribeRun>);
  setExecutionOutputs([
    {
      id: "execution-1",
      nodeId: "agent",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      outputs: {
        failed: [{ type: "runnerCodex.finished", data: { status: "failed", exit_code: 1, error: FAILURE_MESSAGE } }],
      },
    },
  ]);
  vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false }));
});

it.each(["Planning", "Implement", "Verify"])("shows the stored Codex error in %s without runner logs", async (name) => {
  renderRun(name);

  expect(await screen.findByRole("alert")).toHaveTextContent(FAILURE_MESSAGE);
  expect(screen.getByRole("link", { name: "View run details" })).toHaveAttribute(
    "href",
    `${RUN_HREF}?activity=execution-1`,
  );
  expect(useEventExecutions).toHaveBeenCalledWith("app-codex", "event-1");
});

it("does not show a stored failure from another execution of the same runner", () => {
  setExecutionOutputs([
    {
      id: "execution-before-retry",
      nodeId: "agent",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      outputs: { failed: [{ data: { error: FAILURE_MESSAGE } }] },
    },
  ]);

  renderRun("Implement");

  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});

it.each(["passed", "cancelled"] as const)("does not show runtime errors when the run is %s", (status) => {
  renderRun("Implement", status);

  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(useEventExecutions).toHaveBeenCalledWith("app-codex", null);
});

it("keeps a stored planning error visible when completed tool activity is collapsed", () => {
  render(
    <AgentActivityView
      activity={{
        id: "activity-failed",
        provider: "codex",
        status: "failed",
        sequence: 4,
        truncated: false,
        items: [
          {
            type: "tool",
            id: "read",
            kind: "bash",
            name: "Bash",
            input: "cat README.md",
            output: "# Example",
            outputStreams: [],
            status: "passed",
            truncated: false,
          },
          { type: "notice", id: "error", code: "codex_error", text: FAILURE_MESSAGE },
        ],
      }}
    />,
  );

  expect(screen.getByRole("alert")).toHaveTextContent(FAILURE_MESSAGE);
  expect(screen.queryByText("cat README.md")).not.toBeInTheDocument();
});

function setExecutionOutputs(executions: CanvasesCanvasNodeExecution[]) {
  vi.mocked(useEventExecutions).mockReturnValue({
    data: { executions },
    isLoading: false,
    isError: false,
  } as unknown as ReturnType<typeof useEventExecutions>);
}

function renderRun(name: string, status: "failed" | "passed" | "cancelled" = "failed") {
  return render(
    <MemoryRouter>
      <LiveAgentSteps
        stage={{ ...implementStage(), id: name.toLowerCase(), name, status, agentSteps: [] }}
        phase={{ ...implementPhase(), id: name.toLowerCase(), name, status, appId: "app-codex", runId: "run-1" }}
        organizationId="org-1"
        emptyNote="No steps for this run."
        runHref={RUN_HREF}
      />
    </MemoryRouter>,
  );
}
