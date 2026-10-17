import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, mock } from "bun:test";

import type { SplitRunPhase } from "../splitRunMocks";
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

const useEventExecutions = mock((_canvasId?: string, _eventId?: string | null) => ({
  data: undefined as { executions?: unknown } | undefined,
  isError: false,
  isLoading: false,
}));

mock.module("../useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas,
}));

mock.module("@/hooks/useCanvasData", () => ({
  ...canvasData,
  useEventExecutions,
}));

const { AgentRunsPage } = await import("./consoleAgentRuns");

const COMMIT_URL = "https://github.com/acme/app/commit/3fc0c4c0123456789abcdef";
const WAIT_NODE_ID = "wait-custom";

function checksRun(name: string, description?: string): AutomationStage {
  return {
    id: "checks-1",
    name,
    componentName: "Fix pull request checks",
    status: "running",
    statusLabel: "Running",
    duration: "2m",
    description,
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
    id: "checks-1",
    name: "Fix pull request checks",
    status: "running",
    duration: "2m",
    componentName: "Fix pull request checks",
    artifacts: [],
    stream: [],
    canvasSteps: [],
    appId: "canvas-checks",
    runId: "run-checks",
  };
}

function mockChecksCanvas(nodes: Array<{ id: string; component: string; configuration?: unknown }>) {
  useSplitRunLiveCanvas.mockReturnValue({
    enabled: true,
    isError: false,
    isLoading: false,
    canvas: {
      key: "implementation",
      title: "Fix pull request checks",
      nodes,
      edges: [],
      statuses: {},
      metrics: {},
    },
    stream: [],
    rootEventId: "event-root",
  });
}

function mockExecutions(executions: unknown) {
  useEventExecutions.mockImplementation((canvasId: string | undefined, eventId: string | null | undefined) => {
    if (canvasId !== "canvas-checks" || eventId !== "event-root") {
      return { data: undefined, isError: false, isLoading: false };
    }
    return { data: { executions }, isError: false, isLoading: false };
  });
}

function renderChecksCard(run: AutomationStage) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AgentRunsPage
        runs={[run]}
        phases={[{ ...checksPhase(), id: run.id }]}
        automationName="Fix pull request checks"
        organizationId="org-1"
      />
    </QueryClientProvider>,
  );
}

function waitExecution(state: string, selectedBuilds: unknown) {
  return [{ nodeId: WAIT_NODE_ID, state, metadata: { selectedBuilds } }];
}

describe("AgentRunsPage Bitbucket pull request checks", () => {
  beforeEach(() => {
    mockChecksCanvas([{ id: WAIT_NODE_ID, component: "bitbucket.waitForBuilds" }]);
    mockExecutions([]);
  });

  it("lists a waiting Bitbucket build as a rich pending row", () => {
    mockExecutions(
      waitExecution("STATE_STARTED", [
        {
          key: "ci/circleci",
          name: "ci/circleci: backend",
          status: "pending",
          description: "The build is pending on CircleCI.",
          detailsUrl: "https://circleci.com/build/1",
        },
      ]),
    );

    renderChecksCard(checksRun(`Waiting for checks on [3fc0c4c](${COMMIT_URL})`, "Waiting for checks on the commit."));

    const row = screen.getByRole("listitem");
    expect(row.querySelector("svg")).toBeTruthy();
    expect(
      screen.getByRole("link", { name: "ci/circleci: backend: The build is pending on CircleCI." }),
    ).toHaveAttribute("href", "https://circleci.com/build/1");
    expect(row).toHaveTextContent("Pending");
    expect(screen.getByText("Waiting for checks on the commit.")).toBeInTheDocument();
  });

  it("lists configured Bitbucket keys as pending when builds are not posted", () => {
    mockChecksCanvas([
      {
        id: WAIT_NODE_ID,
        component: "bitbucket.waitForBuilds",
        configuration: { buildKeys: ["ci/circleci: backend"] },
      },
    ]);
    mockExecutions(waitExecution("STATE_STARTED", []));

    renderChecksCard(checksRun(`Waiting for checks on [3fc0c4c](${COMMIT_URL})`));

    expect(screen.getAllByRole("listitem").map((item) => item.textContent)).toEqual(["ci/circleci: backendPending"]);
  });

  it("shows passed Bitbucket rows and hides the matching markdown list", () => {
    mockExecutions(
      waitExecution("STATE_FINISHED", [
        {
          name: "ci/circleci: backend",
          status: "completed",
          conclusion: "success",
          description: "Your tests passed on CircleCI!",
          detailsUrl: "https://circleci.com/build/1",
        },
      ]),
    );

    renderChecksCard(
      checksRun(
        `Checks passed on [3fc0c4c](${COMMIT_URL})`,
        "· [ci/circleci: backend: Your tests passed on CircleCI!](https://circleci.com/build/1)",
      ),
    );

    const row = screen.getByRole("listitem");
    expect(row.querySelector("svg")).toBeTruthy();
    expect(screen.getByRole("link", { name: "ci/circleci: backend: Your tests passed on CircleCI!" })).toHaveAttribute(
      "href",
      "https://circleci.com/build/1",
    );
    expect(row).toHaveTextContent("Passed");
    expect(screen.queryByTestId("redesign-run-description")).not.toBeInTheDocument();
  });

  it("shows failed and stopped Bitbucket rows and hides a Failed builds list", () => {
    mockExecutions(
      waitExecution("STATE_FINISHED", [
        {
          name: "pipeline",
          status: "completed",
          conclusion: "failed",
          description: "backend",
          detailsUrl: "https://example.com/pipeline",
          summary: "Your tests failed on CircleCI!",
        },
        {
          key: "deploy",
          name: "deploy",
          status: "completed",
          conclusion: "stopped",
          detailsUrl: "https://example.com/deploy",
        },
      ]),
    );

    renderChecksCard(
      checksRun(
        `Fixing failed checks on [3fc0c4c](${COMMIT_URL})`,
        "Failed builds\n· [pipeline: backend](https://example.com/pipeline): Your tests failed on CircleCI!\n· [deploy](https://example.com/deploy)",
      ),
    );

    expect(screen.getAllByRole("listitem").map((item) => item.textContent)).toEqual([
      "pipeline: backendYour tests failed on CircleCI!Failed",
      "deployFailed",
    ]);
    expect(screen.getByRole("link", { name: "pipeline: backend" })).toHaveAttribute(
      "href",
      "https://example.com/pipeline",
    );
    expect(screen.queryByText("Failed builds")).not.toBeInTheDocument();
    expect(screen.queryByTestId("redesign-run-description")).not.toBeInTheDocument();
  });

  it("uses the build key when the Bitbucket build name is empty", () => {
    mockExecutions(
      waitExecution("STATE_FINISHED", [
        {
          key: "ci/circleci",
          name: " ",
          status: "completed",
          conclusion: "success",
          detailsUrl: "https://example.com/ci",
        },
      ]),
    );

    renderChecksCard(checksRun(`Checks passed on [3fc0c4c](${COMMIT_URL})`));

    expect(screen.getByRole("link", { name: "ci/circleci" })).toHaveAttribute("href", "https://example.com/ci");
    expect(screen.getByRole("listitem")).toHaveTextContent("Passed");
  });
});
