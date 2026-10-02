import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

const LONG_COMMENT = [
  "## Embedded files block duplication",
  "When a task description contains an embedded file reference, this request sends the original file ID unchanged.",
  "Task creation then tries to bind that file to the new task.",
  "The server rejects it because it belongs to the original task.",
  "The description file references need handling even if attachments are not meant to be copied.",
  "## Duplicate ignores create permission",
  "A user who can read tasks but cannot create them still sees an active Duplicate button.",
].join("\n\n");

function commentRun(description: string): AutomationStage {
  return {
    id: "address-1",
    name: "@greptile-apps[bot] left a review",
    componentName: "Address PR feedback",
    status: "passed",
    statusLabel: "Passed",
    duration: "31m 9s",
    description,
    checks: [],
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
      <AgentRunsPage runs={runs} phases={[]} automationName="Address PR feedback" />
    </QueryClientProvider>,
  );
}

function stubElementHeights({ scrollHeight, clientHeight }: { scrollHeight: number; clientHeight: number }) {
  Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, value: scrollHeight });
  Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, value: clientHeight });
  return () => {
    delete (HTMLElement.prototype as unknown as { scrollHeight?: number }).scrollHeight;
    delete (HTMLElement.prototype as unknown as { clientHeight?: number }).clientHeight;
  };
}

describe("AgentRunsPage run comments", () => {
  it("clips a long comment to five lines and expands it on Show more", async () => {
    const restoreHeights = stubElementHeights({ scrollHeight: 240, clientHeight: 100 });
    try {
      const user = userEvent.setup();
      renderRuns([commentRun(LONG_COMMENT)]);

      const body = screen.getByTestId("redesign-run-description-body");
      expect(body).toHaveClass("line-clamp-5");
      expect(screen.getByRole("button", { name: "Show more" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "Show more" }));

      expect(body).not.toHaveClass("line-clamp-5");
      expect(screen.getByRole("button", { name: "Show less" })).toBeInTheDocument();
    } finally {
      restoreHeights();
    }
  });

  it("does not add Show more when a short comment fits", () => {
    const restoreHeights = stubElementHeights({ scrollHeight: 40, clientHeight: 40 });
    try {
      renderRuns([commentRun("Checks passed on the pull request.")]);

      expect(screen.queryByRole("button", { name: "Show more" })).not.toBeInTheDocument();
    } finally {
      restoreHeights();
    }
  });
});

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

function mockChecksCanvas(nodes: Array<{ id: string; component: string }> = []) {
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

function mockExecutions(executions: unknown, failed = false) {
  useEventExecutions.mockImplementation((canvasId, eventId) => {
    if (failed || canvasId !== "canvas-checks" || eventId !== "event-root") {
      return { data: undefined, isError: failed, isLoading: false };
    }
    return { data: { executions }, isError: false, isLoading: false };
  });
}

function renderChecksCard(run: AutomationStage, automationName = "Fix pull request checks") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AgentRunsPage
        runs={[run]}
        phases={[{ ...checksPhase(), id: run.id }]}
        automationName={automationName}
        organizationId="org-1"
      />
    </QueryClientProvider>,
  );
}

describe("AgentRunsPage pull request checks", () => {
  beforeEach(() => {
    mockChecksCanvas([{ id: WAIT_NODE_ID, component: "github.waitForPullRequestChecks" }]);
    mockExecutions([]);
  });

  it("lists watched checks under the commit line while waiting", () => {
    mockExecutions([
      {
        nodeId: "other-node",
        metadata: {
          selectedChecks: [{ name: "decoy-check", status: "completed", conclusion: "failure" }],
        },
      },
      {
        nodeId: WAIT_NODE_ID,
        state: "STATE_STARTED",
        metadata: {
          checks: [{ name: "unwatched", status: "completed", conclusion: "failure" }],
          selectedChecks: [
            { name: "build", status: "in_progress", detailsUrl: "https://github.com/acme/app/runs/1" },
            { name: "lint", status: "completed", conclusion: "failure" },
          ],
        },
      },
    ]);

    renderChecksCard(checksRun(`Waiting for checks on [3fc0c4c](${COMMIT_URL})`, "Waiting for checks on the commit."));

    expect(screen.getByRole("link", { name: "3fc0c4c" })).toHaveAttribute("target", "_blank");
    const rows = screen.getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual(["buildPending", "lintFailed"]);
    expect(screen.getByRole("link", { name: "build" })).toHaveAttribute("href", "https://github.com/acme/app/runs/1");
    expect(screen.getByRole("link", { name: "build" })).toHaveAttribute("target", "_blank");
    expect(screen.queryByText("unwatched")).not.toBeInTheDocument();
    expect(screen.queryByText("decoy-check")).not.toBeInTheDocument();
    expect(screen.getByText("Waiting for checks on the commit.")).toBeInTheDocument();
  });

  it("keeps the waiting sentence when the snapshot has no watched checks", () => {
    mockExecutions([
      {
        nodeId: WAIT_NODE_ID,
        state: "STATE_STARTED",
        metadata: {
          selectedChecks: [],
          checks: [{ name: "build", status: "completed", conclusion: "success" }],
        },
      },
    ]);

    renderChecksCard(checksRun(`Waiting for checks on [3fc0c4c](${COMMIT_URL})`, "Waiting for checks on the commit."));

    expect(screen.getByRole("link", { name: "3fc0c4c" })).toBeInTheDocument();
    expect(screen.getByText("Waiting for checks on the commit.")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Pull request checks" })).not.toBeInTheDocument();
    expect(screen.queryByText("build")).not.toBeInTheDocument();
  });

  it("maps check conclusions with the wait component rules", () => {
    mockExecutions([
      {
        nodeId: WAIT_NODE_ID,
        state: "STATE_FINISHED",
        metadata: {
          selectedChecks: [
            { name: "running", status: "in_progress" },
            { name: "failed", status: "completed", conclusion: "failure" },
            { name: "errored", status: "completed", conclusion: "error" },
            { name: "timed out", status: "completed", conclusion: "timed_out" },
            { name: "action required", status: "completed", conclusion: "action_required" },
            { name: "unknown", status: "completed", conclusion: "startup_failure" },
            { name: "succeeded", status: "completed", conclusion: "success" },
            { name: "neutral", status: "completed", conclusion: "neutral" },
            { name: "skipped", status: "completed", conclusion: "skipped" },
            { name: "cancelled", status: "completed", conclusion: "cancelled" },
            { name: "empty", status: "completed", conclusion: "" },
          ],
        },
      },
    ]);

    renderChecksCard(checksRun(`Checks passed on [3fc0c4c](${COMMIT_URL})`));

    const labels = screen.getAllByRole("listitem").map((row) => row.textContent);
    expect(labels).toEqual([
      "runningPending",
      "failedFailed",
      "erroredFailed",
      "timed outFailed",
      "action requiredFailed",
      "unknownPending",
      "succeededPassed",
      "neutralPassed",
      "skippedPassed",
      "cancelledPassed",
      "emptyPassed",
    ]);
  });

  it("shows the same rows after checks pass and hides the markdown list", () => {
    mockExecutions([
      {
        nodeId: WAIT_NODE_ID,
        state: "STATE_FINISHED",
        metadata: {
          selectedChecks: [
            { name: "build", status: "completed", conclusion: "success", detailsUrl: "https://example.com/build" },
            { name: "lint", status: "completed", conclusion: "success" },
          ],
        },
      },
    ]);

    renderChecksCard(
      checksRun(`Checks passed on [3fc0c4c](${COMMIT_URL})`, "· [build: CI](https://example.com/build)\n· lint"),
    );

    expect(screen.getByText(/Checks passed on/)).toBeInTheDocument();
    expect(screen.getAllByRole("listitem").map((row) => row.textContent)).toEqual(["buildPassed", "lintPassed"]);
    expect(screen.queryByTestId("redesign-run-description")).not.toBeInTheDocument();
    expect(screen.queryByText("build: CI")).not.toBeInTheDocument();
  });

  it("shows the same rows after checks fail and hides the markdown list", () => {
    mockExecutions([
      {
        nodeId: WAIT_NODE_ID,
        state: "STATE_FINISHED",
        metadata: {},
        outputs: {
          failed: [
            {
              data: {
                selectedChecks: [
                  {
                    name: "build",
                    status: "completed",
                    conclusion: "failure",
                    detailsUrl: "https://example.com/build",
                  },
                ],
                checks: [{ name: "unwatched", status: "completed", conclusion: "failure" }],
              },
            },
          ],
        },
      },
    ]);

    renderChecksCard(
      checksRun(
        `Fixing failed checks on [3fc0c4c](${COMMIT_URL})`,
        "Failed checks\n· [build: CI](https://example.com/build): The build failed on Semaphore 2.0.",
      ),
    );

    expect(screen.getByText(/Fixing failed checks on/)).toBeInTheDocument();
    expect(screen.getAllByRole("listitem").map((row) => row.textContent)).toEqual(["buildFailed"]);
    expect(screen.queryByText("Failed checks")).not.toBeInTheDocument();
    expect(screen.queryByText(/The build failed on Semaphore/)).not.toBeInTheDocument();
    expect(screen.queryByText("unwatched")).not.toBeInTheDocument();
  });

  it("keeps the description when the execution read fails", () => {
    mockExecutions([], true);

    renderChecksCard(checksRun(`Waiting for checks on [3fc0c4c](${COMMIT_URL})`, "Waiting for checks on the commit."));

    expect(screen.getByText("Waiting for checks on the commit.")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Pull request checks" })).not.toBeInTheDocument();
  });

  it("does not use emitted checks while the wait is still running", () => {
    mockExecutions([
      {
        nodeId: WAIT_NODE_ID,
        state: "STATE_STARTED",
        metadata: { checks: [{ name: "unwatched", status: "in_progress" }] },
        outputs: {
          passed: [{ data: { selectedChecks: [{ name: "build", status: "completed", conclusion: "success" }] } }],
        },
      },
    ]);

    renderChecksCard(checksRun(`Waiting for checks on [3fc0c4c](${COMMIT_URL})`, "Waiting for checks on the commit."));

    expect(screen.getByText("Waiting for checks on the commit.")).toBeInTheDocument();
    expect(screen.queryByText("build")).not.toBeInTheDocument();
  });

  it("leaves an address-feedback card unchanged", () => {
    mockChecksCanvas([{ id: "comment", component: "github.onPullRequestComment" }]);
    useEventExecutions.mockReturnValue({
      data: {
        executions: [
          {
            nodeId: "comment",
            metadata: { selectedChecks: [{ name: "build", status: "completed", conclusion: "success" }] },
          },
        ],
      },
      isError: false,
      isLoading: false,
    });

    renderChecksCard(commentRun("Please add tests for the new card."), "Address PR feedback");

    expect(screen.getByText("Please add tests for the new card.")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Pull request checks" })).not.toBeInTheDocument();
    expect(screen.queryByText("build")).not.toBeInTheDocument();
  });
});
