import type { CanvasesCanvasNodeExecution } from "@/api-client";
import * as canvasData from "@/hooks/useCanvasData";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { expect, it, mock } from "bun:test";

import type { SplitRunPhase } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";

const executions = [
  {
    id: "queued-execution",
    nodeId: "queued-agent",
    state: "STATE_STARTED",
    runnerJob: { id: "job-1", state: "queued" },
  },
  {
    id: "starting-execution",
    nodeId: "starting-agent",
    state: "STATE_STARTED",
    runnerJob: { id: "job-2", state: "reserved" },
  },
  {
    id: "running-execution",
    nodeId: "running-agent",
    state: "STATE_STARTED",
    runnerJob: { id: "job-3", state: "running" },
  },
] satisfies CanvasesCanvasNodeExecution[];

mock.module("@/hooks/useCanvasData", () => ({
  ...canvasData,
  useEventExecutions: () => ({ data: { executions }, isError: false, isLoading: false }),
}));

mock.module("../useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () => ({
    enabled: true,
    isError: false,
    isLoading: false,
    canvas: {
      key: "implementation",
      title: "Verify changes",
      nodes: [],
      edges: [],
      statuses: {},
      metrics: {},
    },
    stream: [],
    rootEventId: "event-root",
  }),
}));

const { AgentRunsPage } = await import("./consoleAgentRuns");

it("shows individual runner waits, preparation, and running work together on desktop and phone activity", () => {
  const run: AutomationStage = {
    id: "verify-1",
    name: "Verify changes",
    componentName: "Verify changes",
    status: "running",
    statusLabel: "Running",
    duration: "2m",
    checks: [],
    outputs: { pullRequests: [], artifacts: [] },
    plumbing: [],
    agentSteps: [],
    steps: [],
    rawLog: "",
  };
  const phase: SplitRunPhase = {
    id: run.id,
    name: run.name,
    status: "running",
    duration: "2m",
    componentName: run.componentName,
    artifacts: [],
    stream: [],
    canvasSteps: [],
    appId: "canvas-verify",
    runId: "run-verify",
  };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <AgentRunsPage runs={[run]} phases={[phase]} automationName={run.name} organizationId="org-1" />
    </QueryClientProvider>,
  );

  const jobs = screen.getByRole("list", { name: "Runner jobs" });
  expect(within(jobs).getByText("queued-agent: Waiting for runner")).toBeVisible();
  expect(within(jobs).getByText("starting-agent: Starting runner")).toBeVisible();
  expect(within(jobs).getByText("running-agent: Running")).toBeVisible();
});
