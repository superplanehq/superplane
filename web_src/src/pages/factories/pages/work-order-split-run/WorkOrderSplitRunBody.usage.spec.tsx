import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import type { ComponentProps } from "react";
import { WorkOrderSplitRunBody } from "./WorkOrderSplitRunBody";
import { SPLIT_RUN_RUNNING } from "./splitRunRunningFixture";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";
import {
  reduceAgentTelemetryRecords,
  type AgentRunTelemetry,
  type AgentPromptUsageSeries,
} from "@/lib/agentRunTelemetry";

let streamResult: Omit<ReturnType<typeof idleLiveLogStream>, "usageSeries" | "telemetry"> & {
  telemetry: AgentRunTelemetry;
  usageSeries: AgentPromptUsageSeries[];
} = idleLiveLogStream(vi.fn());
vi.mock("./useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () => ({ enabled: false, isError: false, stream: [] }),
}));
vi.mock("./useSplitRunStreamArtifacts", () => ({
  useSplitRunStreamArtifacts: () => ({
    byNodeId: new Map(),
    byNodeName: new Map(),
    byRunId: new Map(),
    pullRequestsByNodeId: new Map(),
    pullRequestsByNodeName: new Map(),
  }),
}));
vi.mock("@/hooks/useSelectableLLMModels", () => ({ useSelectableLLMModels: () => ({ data: [] }) }));
vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: () => streamResult,
}));

it("keeps execution tokens live after the task body collapses its logs", async () => {
  const report = (total: number) => {
    const telemetry = reduceAgentTelemetryRecords([
      {
        type: "turn",
        turn: 1,
        usage_state: "measured",
        usage: { total_tokens: total, input_tokens: 10, cache_read_input_tokens: total - 10 },
      },
    ]);
    streamResult = { ...idleLiveLogStream(vi.fn()), telemetry, usageSeries: [{ name: "Codex", telemetry }] };
  };
  report(210);
  const props: ComponentProps<typeof WorkOrderSplitRunBody> = {
    organizationId: "org-1",
    canUpdate: false,
    fixture: {
      ...SPLIT_RUN_RUNNING,
      currentPhaseId: "plan",
      phases: [
        {
          ...PHASE,
          status: "running",
          canvasKey: null,
          appId: "canvas",
          runId: "run",
          stream: [
            line({
              id: "agent",
              componentName: "Codex",
              component: "runnerCodex",
              executionId: "execution",
              status: "running",
            }),
          ],
        },
      ],
    },
    footerActions: {
      busy: false,
      handleStop: async () => undefined,
      handleStopAutomation: async () => undefined,
      handleReject: async () => undefined,
      handleArchive: async () => undefined,
      handleContinueFixes: async () => undefined,
    },
    follow: {
      following: true,
      scrollRef: { current: null },
      onScroll: () => undefined,
      showJumpToLatest: false,
      setFollowing: () => undefined,
    },
    onStreamTick: () => undefined,
  };
  const view = render(
    <MemoryRouter>
      <WorkOrderSplitRunBody {...props} />
    </MemoryRouter>,
  );
  await waitFor(() => expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("210"));
  const toggle = screen.getByRole("button", { name: "Plan" });
  fireEvent.click(toggle);
  expect(toggle).toHaveAttribute("aria-expanded", "false");
  report(420);
  view.rerender(
    <MemoryRouter>
      <WorkOrderSplitRunBody {...props} />
    </MemoryRouter>,
  );
  await waitFor(() => expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("420"));
});
