import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, mock } from "bun:test";

import type { AutomationStage } from "./automationsViewModel";
import { FEATURE_FACTORY_MULTI_REPO_TASKS } from "@/lib/experimentalFeatures";

mock.module("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => id === FEATURE_FACTORY_MULTI_REPO_TASKS,
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  }),
}));

mock.module("../useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () => ({
    enabled: false,
    isError: false,
    isLoading: false,
    canvas: undefined,
    stream: [],
    rootEventId: undefined,
  }),
}));

const { AgentRunsPage } = await import("./consoleAgentRuns");

function mergeConfidenceRun(id: string, repository: string): AutomationStage {
  return {
    id,
    name: "Merge confidence",
    componentName: "Merge confidence",
    status: "passed",
    statusLabel: "Passed",
    duration: "2m",
    checks: [],
    repository,
    outputs: { pullRequests: [], artifacts: [] },
    plumbing: [],
    agentSteps: [],
    steps: [],
    rawLog: "",
  };
}

function renderRuns(runs: AutomationStage[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AgentRunsPage runs={runs} phases={[]} automationName="Merge confidence" />
    </QueryClientProvider>,
  );
}

describe("AgentRunsPage merge confidence repositories", () => {
  it("names the repository of each run when the runs score several repositories", () => {
    renderRuns([mergeConfidenceRun("merge-api", "acme/api"), mergeConfidenceRun("merge-docs", "acme/docs")]);

    expect(screen.getByRole("button", { name: "Toggle Merge confidence for acme/api" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Toggle Merge confidence for acme/docs" })).toBeInTheDocument();
  });

  it("does not repeat the repository when every run scores the same one", () => {
    renderRuns([mergeConfidenceRun("merge-1", "acme/api"), mergeConfidenceRun("merge-2", "acme/api")]);

    expect(screen.queryByTestId("agent-run-repository")).not.toBeInTheDocument();
  });
});
