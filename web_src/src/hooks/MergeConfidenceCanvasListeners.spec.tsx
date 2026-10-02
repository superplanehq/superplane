import type { CanvasesCanvasRun } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const {
  canvasesDescribeRun,
  canvasesListRuns,
  factoriesListFactoryAutomations,
  factoriesListWorkOrderEvents,
  useCanvasWebsocket,
} = vi.hoisted(() => ({
  canvasesDescribeRun: vi.fn(),
  canvasesListRuns: vi.fn(),
  factoriesListFactoryAutomations: vi.fn(),
  factoriesListWorkOrderEvents: vi.fn(),
  useCanvasWebsocket: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  canvasesDescribeRun,
  canvasesListRuns,
  factoriesListFactoryAutomations,
  factoriesListWorkOrderEvents,
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasWebsocket,
}));

import { MergeConfidenceCanvasListeners } from "./MergeConfidenceCanvasListeners";
import { useFactoryMergeConfidenceRuns } from "./useMergeConfidenceRuns";

function scoreRun(id: string, number: number, repository: string): CanvasesCanvasRun {
  return {
    id,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-08-26T11:00:00Z",
    rootEvent: {
      data: {
        type: "github.pullRequest",
        data: {
          number,
          pull_request: {
            number,
            base: { repo: { full_name: repository } },
          },
          repository: { full_name: repository },
        },
      },
    },
  };
}

const pullRequests = [{ number: "12", repository: "acme/app", url: "https://github.com/acme/app/pull/12" }];

function TaskLog() {
  const mergeConfidence = useFactoryMergeConfidenceRuns("org-1", "factory-1", pullRequests, "order-1");
  return createElement(
    "div",
    null,
    createElement(MergeConfidenceCanvasListeners, {
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: "order-1",
      canvasIds: mergeConfidence.canvasIds,
      taskKey: mergeConfidence.taskKey,
      pullRequests,
    }),
    createElement("span", { "data-testid": "runs" }, mergeConfidence.runs.map((entry) => entry.run.id).join(",")),
  );
}

function renderTaskLog() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(createElement(QueryClientProvider, { client: queryClient }, createElement(TaskLog)));
  return { ...view, queryClient };
}

describe("MergeConfidenceCanvasListeners", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    factoriesListFactoryAutomations.mockResolvedValue({
      data: { automations: [{ id: "app-merge", name: "Merge confidence", columnKey: "verify" }] },
    });
    factoriesListWorkOrderEvents.mockResolvedValue({ data: { events: [], totalCount: 0, hasNextPage: false } });
    canvasesListRuns.mockResolvedValue({
      data: {
        runs: Array.from({ length: 25 }, (_, index) => scoreRun(`run-other-${index}`, 99, "acme/other")),
        totalCount: 50,
        hasNextPage: true,
      },
    });
    canvasesDescribeRun.mockResolvedValue({ data: { run: scoreRun("run-missed", 12, "acme/app") } });
  });

  it("loads a score missed during disconnect after the socket reconnects", async () => {
    const { getByTestId } = renderTaskLog();

    await waitFor(() => expect(useCanvasWebsocket).toHaveBeenCalled());
    await waitFor(() => expect(factoriesListWorkOrderEvents).toHaveBeenCalledTimes(1));
    expect(getByTestId("runs").textContent).toBe("");

    factoriesListWorkOrderEvents.mockResolvedValue({
      data: {
        events: [
          {
            type: "order.check.reported",
            event: { run: { id: "run-missed" }, automation: { appId: "app-merge" } },
          },
        ],
        totalCount: 1,
        hasNextPage: false,
      },
    });

    const onConnectionOpen = useCanvasWebsocket.mock.calls.at(-1)?.[0]?.onConnectionOpen as (() => void) | undefined;
    act(() => {
      onConnectionOpen?.();
    });

    await waitFor(() => expect(getByTestId("runs").textContent).toBe("run-missed"));
    expect(canvasesDescribeRun).toHaveBeenCalledWith(
      expect.objectContaining({ path: { canvasId: "app-merge", runId: "run-missed" } }),
    );
  });
});
